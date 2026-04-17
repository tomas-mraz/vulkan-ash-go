package ash

import (
	"fmt"
	"log/slog"
	"runtime"

	vk "github.com/tomas-mraz/vulkan"
)

// Frame is a per-frame record handed to Renderer.Draw.
//
// Cmd is already reset and begun by the Session; Draw just records commands
// into it. The Session ends and submits the command buffer after Draw returns.
type Frame struct {
	Cmd        vk.CommandBuffer
	ImageIndex uint32
	Extent     vk.Extent2D
	Swapchain  *Swapchain
}

// Renderer is the application-supplied object that owns all size-dependent and
// size-independent graphics resources and records frame commands.
//
// Lifecycle, as driven by Session.Run:
//
//	CreateOnce(s)                                   // device is live
//	CreateSized(s, extent0)                         // first swapchain built
//	    Draw(s, f)   ...   Draw(s, f)               // steady state
//	    [rotation / resize observed]
//	    DestroySized()
//	    CreateSized(s, extentN)                     // new swapchain built
//	    Draw(s, f)   ...
//	[surface lost]
//	DestroySized()
//	DestroyOnce()                                   // device torn down
//
// All methods run on the Session's run goroutine. DestroySized / DestroyOnce
// must be idempotent — Run calls them in its shutdown path even if CreateSized
// / CreateOnce failed partway through.
type Renderer interface {
	// CreateOnce builds resources that stay valid for the lifetime of the device:
	// textures, descriptor set layouts, static vertex/index buffers, shaders,
	// uniforms whose layout does not depend on swapchain length.
	CreateOnce(s *Session) error

	// DestroyOnce releases CreateOnce resources. Must be idempotent.
	DestroyOnce()

	// CreateSized builds resources that depend on the swapchain extent:
	// depth image, render pass (if its attachments encode extent-linked choices),
	// framebuffers, and graphics pipelines whose viewport/scissor is baked in.
	// Called once per swapchain generation.
	CreateSized(s *Session, extent vk.Extent2D) error

	// DestroySized releases CreateSized resources. Must be idempotent.
	DestroySized()

	// Draw records one frame into f.Cmd. The Session has already begun the
	// command buffer; Draw issues CmdBeginRenderPass, draw calls,
	// CmdEndRenderPass, and returns. Session handles end + submit + present.
	Draw(s *Session, f *Frame) error
}

// SessionOptions configures device creation for NewSession.
type SessionOptions struct {
	DeviceOptions   *DeviceOptions
	EnableTiming    bool // enable VK_GOOGLE_display_timing when available
	FrameSubmitWait uint64
}

// Session owns the whole Vulkan stack (instance, device, swapchain, command
// pool, sync objects) and orchestrates the render loop on top of a Host.
//
// Typical usage:
//
//	host := ash.NewDesktopHost(...)   // or ash.NewAndroidHost(a)
//	sess := ash.NewSession(host, "MyApp", nil)
//	sess.Run(&myRenderer{})
//
// Session.Run is a blocking state machine. It reacts to HostEvents to build
// and tear down Vulkan resources, and drives the Renderer's lifecycle callbacks
// at the corresponding moments.
type Session struct {
	Host    Host
	AppName string
	Opts    SessionOptions

	// Populated while the device is alive.
	Manager       *Manager
	Swapchain     *Swapchain
	Ctx           *SwapchainContext
	CmdCtx        *CommandContext
	Sync          *SyncInfo
	DisplayTiming *DisplayTiming

	// running is true between "device + swapchain built" and "surface lost".
	running bool
}

// NewSession returns a Session bound to the given Host. NewSession does not
// touch the platform; all platform I/O happens inside Run.
func NewSession(host Host, appName string, opts *SessionOptions) *Session {
	if opts == nil {
		opts = &SessionOptions{EnableTiming: true}
	}
	return &Session{
		Host:    host,
		AppName: appName,
		Opts:    *opts,
	}
}

// Run starts the platform Host, drives the Renderer lifecycle, and blocks
// until HostEventClose is received or a fatal error occurs.
func (s *Session) Run(r Renderer) error {
	if err := s.Host.Start(); err != nil {
		return fmt.Errorf("host.Start: %w", err)
	}
	defer s.Host.Shutdown()

	events := s.Host.Events()
	for {
		if !s.running {
			// Idle path: nothing to render, block on the next event.
			select {
			case ev, ok := <-events:
				if !ok {
					// Channel closed (Android OnDestroy) — exit cleanly.
					return nil
				}
				done, err := s.handleEvent(ev, r)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
			}
			continue
		}

		// Active path: drain any pending events without blocking, pump the
		// platform, then render one frame. glfw.PollEvents must run on the
		// main goroutine, which is where Run is executing.
		drainLoop:
		for {
			select {
			case ev, ok := <-events:
				if !ok {
					s.teardownDevice(r)
					return nil
				}
				done, err := s.handleEvent(ev, r)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
				if !s.running {
					break drainLoop
				}
			default:
				break drainLoop
			}
		}
		s.Host.Pump()
		// Pump may have just enqueued a Close — peek without blocking.
		select {
		case ev, ok := <-events:
			if !ok {
				s.teardownDevice(r)
				return nil
			}
			done, err := s.handleEvent(ev, r)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		default:
		}

		if !s.running {
			continue
		}

		if err := s.renderFrame(r); err != nil {
			slog.Error("Session.renderFrame", "err", err)
			// A single bad frame shouldn't kill the app — drop it and keep going.
			// If the swapchain is fundamentally broken, NeedsRecreate flips and
			// the next iteration rebuilds.
		}
	}
}

// handleEvent translates a HostEvent into Vulkan setup/teardown transitions.
// Returns done=true when the app should exit.
func (s *Session) handleEvent(ev HostEvent, r Renderer) (done bool, err error) {
	switch ev.Kind {
	case HostEventSurfaceAvailable:
		if s.running {
			// A second Available without a Lost in between: treat it like a
			// redraw-needed and force recreation rather than full teardown.
			s.Ctx.RequestRecreate()
			return false, nil
		}
		if err := s.setupDevice(r, ev.Extent); err != nil {
			return false, fmt.Errorf("setupDevice: %w", err)
		}
		return false, nil

	case HostEventSurfaceLost:
		s.teardownDevice(r)
		return false, nil

	case HostEventSurfaceInvalidated:
		if s.running {
			s.Ctx.RequestRecreate()
		}
		return false, nil

	case HostEventClose:
		s.teardownDevice(r)
		return true, nil
	}
	return false, nil
}

// setupDevice is the full initial-bringup path:
// InitVulkan → Manager → Swapchain → SwapchainContext → CmdCtx → Sync →
// DisplayTiming → Renderer.CreateOnce → Renderer.CreateSized.
// On any failure all partial state is rolled back and running stays false.
func (s *Session) setupDevice(r Renderer, hint vk.Extent2D) (err error) {
	if err := s.Host.InitVulkan(); err != nil {
		return fmt.Errorf("InitVulkan: %w", err)
	}

	opts := s.Opts.DeviceOptions
	if opts == nil {
		opts = &DeviceOptions{}
	}
	hostExts := s.Host.InstanceExtensions()
	if len(hostExts) > 0 {
		merged := make([]string, 0, len(opts.InstanceExtensions)+len(hostExts))
		merged = append(merged, opts.InstanceExtensions...)
		merged = append(merged, hostExts...)
		optsCopy := *opts
		optsCopy.InstanceExtensions = merged
		opts = &optsCopy
	}

	mgr, err := NewManager(s.AppName, s.Host.CreateSurface, opts)
	if err != nil {
		return fmt.Errorf("NewManager: %w", err)
	}
	s.Manager = &mgr

	// From here on any failure must tear back down through teardownDevice so
	// Vulkan objects aren't leaked. We stage resources on the Session fields
	// as they come up and rely on teardownDevice to walk them in reverse.
	defer func() {
		if err != nil {
			s.teardownDevice(r)
		}
	}()

	swap, err := NewSwapchain(&mgr, hint)
	if err != nil {
		return fmt.Errorf("NewSwapchain: %w", err)
	}
	s.Swapchain = &swap

	ctx := NewSwapchainContext(&mgr, s.Swapchain)
	s.Ctx = &ctx

	cmdCtx, err := NewCommandContext(mgr.Device, 0, s.Swapchain.DefaultSwapchainLen())
	if err != nil {
		return fmt.Errorf("NewCommandContext: %w", err)
	}
	s.CmdCtx = &cmdCtx

	sync, err := NewSyncObjects(mgr.Device)
	if err != nil {
		return fmt.Errorf("NewSyncObjects: %w", err)
	}
	s.Sync = &sync

	if s.Opts.EnableTiming {
		// VK_GOOGLE_display_timing is currently only enabled on Android by
		// Manager; calling its entry points on platforms where it isn't
		// loaded crashes in cgo. Guard on actual device support.
		if ok, _ := CheckDeviceExtensions(mgr.Gpu, []string{vk.GoogleDisplayTimingExtensionName}); ok {
			dt := NewDisplayTiming(mgr.Device, s.Swapchain.DefaultSwapchain())
			s.DisplayTiming = &dt
			s.Ctx.SetDisplayTiming(s.DisplayTiming)
		}
	}

	if err := r.CreateOnce(s); err != nil {
		return fmt.Errorf("renderer.CreateOnce: %w", err)
	}
	if err := r.CreateSized(s, s.Swapchain.DisplaySize); err != nil {
		// CreateOnce succeeded but CreateSized didn't; make sure DestroyOnce
		// is still called during teardown. Flip running briefly so teardown
		// knows the renderer has CreateOnce state live.
		s.running = true
		return fmt.Errorf("renderer.CreateSized: %w", err)
	}

	s.running = true
	return nil
}

// teardownDevice reverses setupDevice, tolerating partial state. Safe to call
// on any failure path or on any number of HostEventSurfaceLost events.
func (s *Session) teardownDevice(r Renderer) {
	if s.Manager != nil {
		// Best-effort wait so we don't destroy resources mid-flight.
		vk.DeviceWaitIdle(s.Manager.Device)
	}
	if s.running {
		r.DestroySized()
		r.DestroyOnce()
		s.running = false
	}
	if s.Sync != nil {
		s.Sync.Destroy()
		s.Sync = nil
	}
	if s.CmdCtx != nil {
		s.CmdCtx.Destroy()
		s.CmdCtx = nil
	}
	if s.Swapchain != nil {
		s.Swapchain.Destroy()
		s.Swapchain = nil
	}
	s.Ctx = nil
	s.DisplayTiming = nil
	if s.Manager != nil {
		s.Manager.Destroy()
		s.Manager = nil
	}
}

// renderFrame drives one iteration: optional recreate, acquire, BeginFrame,
// Renderer.Draw, EndFrame, submit, present.
func (s *Session) renderFrame(r Renderer) error {
	// Before acquire: honor any pending recreation request.
	if s.Ctx.NeedsRecreate() {
		if err := s.recreateSwapchain(r); err != nil {
			return fmt.Errorf("recreateSwapchain(pre): %w", err)
		}
	}

	imageIndex, acquired, err := s.Ctx.AcquireNextImage(vk.MaxUint64, s.Sync.Semaphore, vk.NullFence)
	if err != nil {
		return fmt.Errorf("AcquireNextImage: %w", err)
	}
	if !acquired {
		// Out-of-date reported during acquire: rebuild now; render next tick.
		if err := s.recreateSwapchain(r); err != nil {
			return fmt.Errorf("recreateSwapchain(acquire): %w", err)
		}
		return nil
	}

	cmd, err := s.Ctx.BeginFrame(imageIndex, s.CmdCtx)
	if err != nil {
		return fmt.Errorf("BeginFrame: %w", err)
	}

	frame := &Frame{
		Cmd:        cmd,
		ImageIndex: imageIndex,
		Extent:     s.Swapchain.DisplaySize,
		Swapchain:  s.Swapchain,
	}
	if err := r.Draw(s, frame); err != nil {
		// Still close the command buffer cleanly so Vulkan doesn't complain.
		_ = s.Ctx.EndFrame(cmd)
		return fmt.Errorf("renderer.Draw: %w", err)
	}

	if err := s.Ctx.EndFrame(cmd); err != nil {
		return fmt.Errorf("EndFrame: %w", err)
	}
	if err := s.Ctx.SubmitRender(cmd, s.Sync.Fence, []vk.Semaphore{s.Sync.Semaphore}); err != nil {
		return fmt.Errorf("SubmitRender: %w", err)
	}
	if _, err := s.Ctx.PresentImage(imageIndex, nil); err != nil {
		return fmt.Errorf("PresentImage: %w", err)
	}
	return nil
}

// recreateSwapchain runs the full sized-resource rebuild:
//   - wait device idle
//   - Renderer.DestroySized
//   - build new swapchain (keeps old handle as OldSwapchain)
//   - rebuild command context for the new swapchain length
//   - Renderer.CreateSized with the new extent
//
// Kept outside the framework's SwapchainContext.Recreate callback API because
// the Session owns more of the state (CmdCtx, Renderer lifecycle) than that
// callback was designed for.
func (s *Session) recreateSwapchain(r Renderer) error {
	if s.Manager == nil || s.Swapchain == nil {
		return fmt.Errorf("recreateSwapchain: session not set up")
	}

	if err := vk.Error(vk.DeviceWaitIdle(s.Manager.Device)); err != nil {
		return fmt.Errorf("DeviceWaitIdle: %w", err)
	}

	r.DestroySized()

	// Ask the host for its current best-guess extent; fall back to what the
	// driver reports via surface capabilities.
	hint, ok := s.Host.CurrentExtent()
	if !ok {
		hint = s.Swapchain.DisplaySize
	}
	hint = s.Manager.QuerySurfaceExtent(hint)

	if err := s.Ctx.Recreate(hint, nil); err != nil {
		return fmt.Errorf("Ctx.Recreate: %w", err)
	}

	// Rebuild cmd context — swapchain length may have changed (it usually
	// doesn't on mainstream Android, but the spec allows it).
	newCmd, err := NewCommandContext(s.Manager.Device, 0, s.Swapchain.DefaultSwapchainLen())
	if err != nil {
		return fmt.Errorf("NewCommandContext: %w", err)
	}
	s.CmdCtx.Destroy()
	*s.CmdCtx = newCmd

	if err := r.CreateSized(s, s.Swapchain.DisplaySize); err != nil {
		return fmt.Errorf("renderer.CreateSized: %w", err)
	}
	return nil
}

// Yield gives other goroutines a chance to run. Called from idle paths to
// avoid hot-spinning while waiting for platform events.
func (s *Session) Yield() {
	runtime.Gosched()
}
