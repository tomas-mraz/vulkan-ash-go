package asch

import (
	"bytes"
	"fmt"
	"math"
	"unsafe"
)

// --- Math helpers ---

func sqrtf(v float32) float32 { return float32(math.Sqrt(float64(v))) }
func sinf(v float32) float32  { return float32(math.Sin(float64(v))) }
func cosf(v float32) float32  { return float32(math.Cos(float64(v))) }
func tanf(v float32) float32  { return float32(math.Tan(float64(v))) }

// DegreesToRadians converts degrees to radians.
func DegreesToRadians(angleDegrees float32) float32 {
	return angleDegrees * float32(math.Pi) / 180.0
}

// RadiansToDegrees converts radians to degrees.
func RadiansToDegrees(angleRadians float32) float32 {
	return angleRadians * 180.0 / float32(math.Pi)
}

// AlignUp rounds size up to the nearest multiple of alignment.
func AlignUp(size, alignment uint32) uint32 {
	return (size + alignment - 1) &^ (alignment - 1)
}

// --- Vec2 ---

type Vec2 [2]float32

func (r *Vec2) Add(a, b *Vec2) {
	for i := 0; i < 2; i++ {
		r[i] = a[i] + b[i]
	}
}

func (r *Vec2) Sub(a, b *Vec2) {
	for i := 0; i < 2; i++ {
		r[i] = a[i] - b[i]
	}
}

func (r *Vec2) Scale(v *Vec2, s float32) {
	for i := 0; i < 2; i++ {
		r[i] = v[i] * s
	}
}

func (v *Vec2) Len() float32 {
	return sqrtf(Vec2MultInner(v, v))
}

func (r *Vec2) Norm(v *Vec2) {
	var k float32 = 1.0 / v.Len()
	r.Scale(v, k)
}

func (r *Vec2) Min(a, b *Vec2) {
	for i := 0; i < 2; i++ {
		if a[i] < b[i] {
			r[i] = a[i]
		} else {
			r[i] = b[i]
		}
	}
}

func (r *Vec2) Max(a, b *Vec2) {
	for i := 0; i < 2; i++ {
		if a[i] > b[i] {
			r[i] = a[i]
		} else {
			r[i] = b[i]
		}
	}
}

func Vec2MultInner(a, b *Vec2) (p float32) {
	for i := 0; i < 2; i++ {
		p += b[i] * a[i]
	}
	return p
}

// --- Vec3 ---

type Vec3 [3]float32

func (r *Vec3) Add(a, b *Vec3) {
	for i := 0; i < 3; i++ {
		r[i] = a[i] + b[i]
	}
}

func (r *Vec3) Sub(a, b *Vec3) {
	for i := 0; i < 3; i++ {
		r[i] = a[i] - b[i]
	}
}

func (r *Vec3) Scale(v *Vec3, s float32) {
	for i := 0; i < 3; i++ {
		r[i] = v[i] * s
	}
}

func (r *Vec3) ScaleVec4(v *Vec4, s float32) {
	for i := 0; i < 3; i++ {
		r[i] = v[i] * s
	}
}

func (r *Vec3) ScaleQuat(q *Quat, s float32) {
	for i := 0; i < 3; i++ {
		r[i] = q[i] * s
	}
}

func (v *Vec3) Len() float32 {
	return sqrtf(Vec3MultInner(v, v))
}

func (r *Vec3) Norm(v *Vec3) {
	var k float32 = 1.0 / v.Len()
	r.Scale(v, k)
}

func (r *Vec3) Min(a, b *Vec3) {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			r[i] = a[i]
		} else {
			r[i] = b[i]
		}
	}
}

func (r *Vec3) Max(a, b *Vec3) {
	for i := 0; i < 3; i++ {
		if a[i] > b[i] {
			r[i] = a[i]
		} else {
			r[i] = b[i]
		}
	}
}

func Vec3MultInner(a, b *Vec3) (p float32) {
	for i := 0; i < 3; i++ {
		p += b[i] * a[i]
	}
	return p
}

func (r *Vec3) MultCross(a, b *Vec3) {
	r[0] = a[1]*b[2] - a[2]*b[1]
	r[1] = a[2]*b[0] - a[0]*b[2]
	r[2] = a[0]*b[1] - a[1]*b[0]
}

func (r *Vec3) Reflect(v, n *Vec3) {
	var p float32 = 2 * Vec3MultInner(v, n)
	for i := 0; i < 3; i++ {
		r[i] = v[i] - p*n[i]
	}
}

func (r *Vec3) QuatMultVec3(q *Quat, v *Vec3) {
	var t = new(Vec3)
	var q_xyz = &Vec3{q[0], q[1], q[2]}
	var u = &Vec3{q[0], q[1], q[2]}
	t.MultCross(q_xyz, v)
	t.Scale(t, 2)
	u.MultCross(q_xyz, t)
	t.Scale(t, q[3])
	r.Add(v, t)
	r.Add(r, u)
}

// --- Vec4 ---

type Vec4 [4]float32

func (r *Vec4) Add(a, b *Vec4) {
	for i := 0; i < 4; i++ {
		r[i] = a[i] + b[i]
	}
}

func (r *Vec4) Sub(a, b *Vec4) {
	for i := 0; i < 4; i++ {
		r[i] = a[i] - b[i]
	}
}

func (r *Vec4) SubVec3(a *Vec4, b *Vec3) {
	for i := 0; i < 3; i++ {
		r[i] = a[i] - b[i]
	}
}

func (r *Vec4) Scale(v *Vec4, s float32) {
	for i := 0; i < 4; i++ {
		r[i] = v[i] * s
	}
}

func (v *Vec4) Len() float32 {
	return sqrtf(Vec4MultInner(v, v))
}

func (r *Vec4) Norm(v *Vec4) {
	var k float32 = 1.0 / v.Len()
	r.Scale(v, k)
}

func (r *Vec4) Min(a, b *Vec4) {
	for i := 0; i < 4; i++ {
		if a[i] < b[i] {
			r[i] = a[i]
		} else {
			r[i] = b[i]
		}
	}
}

func (r *Vec4) Max(a, b *Vec4) {
	for i := 0; i < 4; i++ {
		if a[i] > b[i] {
			r[i] = a[i]
		} else {
			r[i] = b[i]
		}
	}
}

func Vec4MultInner(a, b *Vec4) (p float32) {
	for i := 0; i < 4; i++ {
		p += b[i] * a[i]
	}
	return p
}

func Vec4MultInner3(a, b *Vec4) (p float32) {
	for i := 0; i < 3; i++ {
		p += b[i] * a[i]
	}
	return p
}

func (r *Vec4) MultCross(a, b *Vec4) {
	r[0] = a[1]*b[2] - a[2]*b[1]
	r[1] = a[2]*b[0] - a[0]*b[2]
	r[2] = a[0]*b[1] - a[1]*b[0]
	r[3] = 1
}

func (r *Vec4) Reflect(v, n *Vec4) {
	var p float32 = 2 * Vec4MultInner(v, n)
	for i := 0; i < 4; i++ {
		r[i] = v[i] - p*n[i]
	}
}

func (r *Vec4) Mat4x4Row(m *Mat4x4, i int) {
	for k := 0; k < 4; k++ {
		r[k] = m[k][i]
	}
}

func (r *Vec4) Mat4x4Col(m *Mat4x4, i int) {
	for k := 0; k < 4; k++ {
		r[k] = m[i][k]
	}
}

func (r *Vec4) Mat4x4MultVec4(m *Mat4x4, v Vec4) {
	for j := 0; j < 4; j++ {
		r[j] = 0
		for i := 0; i < 4; i++ {
			r[j] += m[i][j] * v[i]
		}
	}
}

func (r *Vec4) QuatMultVec4(q *Quat, v *Vec4) {
	var t = new(Vec4)
	var q_xyz = &Vec4{q[0], q[1], q[2]}
	var u = &Vec4{q[0], q[1], q[2]}
	t.MultCross(q_xyz, v)
	t.Scale(t, 2)
	u.MultCross(q_xyz, t)
	t.Scale(t, q[3])
	r.Add(v, t)
	r.Add(r, u)
}

// --- Quat ---

type Quat [4]float32

func (q *Quat) Identity() {
	q[0] = 0
	q[1] = 0
	q[2] = 0
	q[3] = 1
}

func (r *Quat) Add(a, b *Quat) {
	for i := 0; i < 4; i++ {
		r[i] = a[i] + b[i]
	}
}

func (r *Quat) AddVec3(a *Quat, v *Vec3) {
	for i := 0; i < 3; i++ {
		r[i] = a[i] + v[i]
	}
}

func (r *Quat) Sub(a, b *Quat) {
	for i := 0; i < 4; i++ {
		r[i] = a[i] - b[i]
	}
}

func (r *Quat) MultCross3(a, b *Quat) {
	r[0] = a[1]*b[2] - a[2]*b[1]
	r[1] = a[2]*b[0] - a[0]*b[2]
	r[2] = a[0]*b[1] - a[1]*b[0]
}

func QuatMultInner3(a, b *Quat) (p float32) {
	for i := 0; i < 3; i++ {
		p += b[i] * a[i]
	}
	return p
}

func (r *Quat) Mult(p, q *Quat) {
	var w = new(Vec3)
	r.MultCross3(p, q)
	w.ScaleQuat(p, q[3])
	r.AddVec3(r, w)
	w.ScaleQuat(q, p[3])
	r.AddVec3(r, w)
	r[3] = p[3]*q[3] - QuatMultInner3(p, q)
}

func (r *Quat) Scale(q *Quat, s float32) {
	for i := 0; i < 4; i++ {
		r[i] = q[i] * s
	}
}

func QuatInnerProduct(a, b *Quat) (p float32) {
	for i := 0; i < 4; i++ {
		p += b[i] * a[i]
	}
	return p
}

func (r *Quat) Conj(q *Quat) {
	for i := 0; i < 3; i++ {
		r[i] = -q[i]
	}
	r[3] = q[3]
}

func (q *Quat) Len() float32 {
	return sqrtf(QuatInnerProduct(q, q))
}

func (r *Quat) Norm(q *Quat) {
	var k float32 = 1.0 / q.Len()
	r.Scale(q, k)
}

func (q *Quat) FromMat4x4(m *Mat4x4) {
	var r float32
	var p = []int{0, 1, 2, 0, 1}
	var idx int
	for i := 0; i < 3; i++ {
		var mm float32 = m[i][i]
		if mm < r {
			continue
		}
		mm = r
		idx = i
	}
	p = p[idx:]
	r = sqrtf(1 + m[p[0]][p[0]] - m[p[1]][p[1]] - m[p[2]][p[2]])
	if r < 1e-6 {
		q[0] = 1
		q[1] = 0
		q[2] = 0
		q[3] = 0
		return
	}
	q[0] = r / 2
	q[1] = (m[p[0]][p[1]] - m[p[1]][p[0]]) / (2 * r)
	q[2] = (m[p[2]][p[0]] - m[p[0]][p[2]]) / (2 * r)
	q[3] = (m[p[2]][p[1]] - m[p[1]][p[2]]) / (2 * r)
}

// --- Mat4x4 ---

type Mat4x4 [4]Vec4

func (m *Mat4x4) Fill(d float32) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			m[i][j] = d
		}
	}
}

func (m *Mat4x4) Identity() {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if i == j {
				m[i][j] = 1
			} else {
				m[i][j] = 0
			}
		}
	}
}

func (m *Mat4x4) Dup(n *Mat4x4) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			m[i][j] = n[i][j]
		}
	}
}

func (m *Mat4x4) Transpose(n *Mat4x4) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			m[i][j] = n[j][i]
		}
	}
}

func (m *Mat4x4) Add(a, b *Mat4x4) {
	for i := 0; i < 4; i++ {
		m[i].Add(&a[i], &b[i])
	}
}

func (m *Mat4x4) Sub(a, b *Mat4x4) {
	for i := 0; i < 4; i++ {
		m[i].Sub(&a[i], &b[i])
	}
}

func (m *Mat4x4) Scale(a *Mat4x4, k float32) {
	for i := 0; i < 4; i++ {
		m[i].Scale(&a[i], k)
	}
}

func (m *Mat4x4) ScaleAniso(a *Mat4x4, x, y, z float32) {
	m[0].Scale(&a[0], x)
	m[1].Scale(&a[1], y)
	m[2].Scale(&a[2], z)
	for i := 0; i < 4; i++ {
		m[3][i] = a[3][i]
	}
}

func (m *Mat4x4) Mult(a, b *Mat4x4) {
	var temp = new(Mat4x4)
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			temp[c][r] = 0
			for k := 0; k < 4; k++ {
				temp[c][r] += a[k][r] * b[c][k]
			}
		}
	}
	m.Dup(temp)
}

func (m *Mat4x4) Translate(x, y, z float32) {
	m.Identity()
	m[3][0] = x
	m[3][1] = y
	m[3][2] = z
}

func (m *Mat4x4) TranslateInPlace(x, y, z float32) {
	var t = &Vec4{x, y, z, 0}
	var r = new(Vec4)
	for i := 0; i < 4; i++ {
		r.Mat4x4Row(m, i)
		m[3][i] += Vec4MultInner(r, t)
	}
}

func (m *Mat4x4) FromVec3MultOuter(a, b *Vec3) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if i < 3 && j < 3 {
				m[i][j] = a[i] * b[j]
			} else {
				m[i][j] = 0
			}
		}
	}
}

func (r *Mat4x4) Rotate(m *Mat4x4, x, y, z, angle float32) {
	var s = sinf(angle)
	var c = cosf(angle)
	var u = &Vec3{x, y, z}
	if u.Len() > 1e-4 {
		u.Norm(u)
		var T = new(Mat4x4)
		T.FromVec3MultOuter(u, u)
		var S = &Mat4x4{
			{0, u[2], -u[1], 0},
			{-u[2], 0, u[0], 0},
			{u[1], -u[0], 0, 0},
			{0, 0, 0, 0},
		}
		S.Scale(S, s)
		var C = new(Mat4x4)
		C.Identity()
		C.Sub(C, T)
		C.Scale(C, c)
		T.Add(T, C)
		T.Add(T, S)
		T[3][3] = 1
		r.Mult(m, T)
	} else {
		r.Dup(m)
	}
}

func (q *Mat4x4) RotateX(m *Mat4x4, angle float32) {
	var s = sinf(angle)
	var c = cosf(angle)
	var R = &Mat4x4{
		{1, 0, 0, 0},
		{0, c, s, 0},
		{0, -s, c, 0},
		{0, 0, 0, 1},
	}
	q.Mult(m, R)
}

func (q *Mat4x4) RotateY(m *Mat4x4, angle float32) {
	var s = sinf(angle)
	var c = cosf(angle)
	var R = &Mat4x4{
		{c, 0, s, 0},
		{0, 1, 0, 0},
		{-s, 0, c, 0},
		{0, 0, 0, 1},
	}
	q.Mult(m, R)
}

func (q *Mat4x4) RotateZ(m *Mat4x4, angle float32) {
	var s = sinf(angle)
	var c = cosf(angle)
	var R = &Mat4x4{
		{c, s, 0, 0},
		{-s, c, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	q.Mult(m, R)
}

func (t *Mat4x4) Invert(m *Mat4x4) {
	var s = new([6]float32)
	s[0] = m[0][0]*m[1][1] - m[1][0]*m[0][1]
	s[1] = m[0][0]*m[1][2] - m[1][0]*m[0][2]
	s[2] = m[0][0]*m[1][3] - m[1][0]*m[0][3]
	s[3] = m[0][1]*m[1][2] - m[1][1]*m[0][2]
	s[4] = m[0][1]*m[1][3] - m[1][1]*m[0][3]
	s[5] = m[0][2]*m[1][3] - m[1][2]*m[0][3]
	var c = new([6]float32)
	c[0] = m[2][0]*m[3][1] - m[3][0]*m[2][1]
	c[1] = m[2][0]*m[3][2] - m[3][0]*m[2][2]
	c[2] = m[2][0]*m[3][3] - m[3][0]*m[2][3]
	c[3] = m[2][1]*m[3][2] - m[3][1]*m[2][2]
	c[4] = m[2][1]*m[3][3] - m[3][1]*m[2][3]
	c[5] = m[2][2]*m[3][3] - m[3][2]*m[2][3]
	var idet float32 = 1.0 / (s[0]*c[5] - s[1]*c[4] + s[2]*c[3] + s[3]*c[2] - s[4]*c[1] + s[5]*c[0])
	t[0][0] = (m[1][1]*c[5] - m[1][2]*c[4] + m[1][3]*c[3]) * idet
	t[0][1] = (-m[0][1]*c[5] + m[0][2]*c[4] - m[0][3]*c[3]) * idet
	t[0][2] = (m[3][1]*s[5] - m[3][2]*s[4] + m[3][3]*s[3]) * idet
	t[0][3] = (-m[2][1]*s[5] + m[2][2]*s[4] - m[2][3]*s[3]) * idet
	t[1][0] = (-m[1][0]*c[5] + m[1][2]*c[2] - m[1][3]*c[1]) * idet
	t[1][1] = (m[0][0]*c[5] - m[0][2]*c[2] + m[0][3]*c[1]) * idet
	t[1][2] = (-m[3][0]*s[5] + m[3][2]*s[2] - m[3][3]*s[1]) * idet
	t[1][3] = (m[2][0]*s[5] - m[2][2]*s[2] + m[2][3]*s[1]) * idet
	t[2][0] = (m[1][0]*c[4] - m[1][1]*c[2] + m[1][3]*c[0]) * idet
	t[2][1] = (-m[0][0]*c[4] + m[0][1]*c[2] - m[0][3]*c[0]) * idet
	t[2][2] = (m[3][0]*s[4] - m[3][1]*s[2] + m[3][3]*s[0]) * idet
	t[2][3] = (-m[2][0]*s[4] + m[2][1]*s[2] - m[2][3]*s[0]) * idet
	t[3][0] = (-m[1][0]*c[3] + m[1][1]*c[1] - m[1][2]*c[0]) * idet
	t[3][1] = (m[0][0]*c[3] - m[0][1]*c[1] + m[0][2]*c[0]) * idet
	t[3][2] = (-m[3][0]*s[3] + m[3][1]*s[1] - m[3][2]*s[0]) * idet
	t[3][3] = (m[2][0]*s[3] - m[2][1]*s[1] + m[2][2]*s[0]) * idet
}

func (r *Mat4x4) OrthoNormalize(m *Mat4x4) {
	r.Dup(m)
	var s float32
	var h = new(Vec3)
	r[2].Norm(&r[2])
	s = Vec4MultInner3(&r[1], &r[2])
	h.ScaleVec4(&r[2], s)
	r[1].SubVec3(&r[1], h)
	r[2].Norm(&r[2])
	s = Vec4MultInner3(&r[1], &r[2])
	h.ScaleVec4(&r[2], s)
	r[1].SubVec3(&r[1], h)
	r[1].Norm(&r[1])
	s = Vec4MultInner3(&r[0], &r[1])
	h.ScaleVec4(&r[1], s)
	r[0].SubVec3(&r[0], h)
	r[0].Norm(&r[0])
}

func (m *Mat4x4) Frustum(l, r, b, t, n, f float32) {
	m[0][0] = 2 * n / (r - l)
	m[0][1] = 0
	m[0][2] = 0
	m[0][3] = 0
	m[1][1] = 2 * n / (t - b)
	m[1][0] = 0
	m[1][2] = 0
	m[1][3] = 0
	m[2][0] = (r + l) / (r - l)
	m[2][1] = (t + b) / (t - b)
	m[2][2] = -(f + n) / (f - n)
	m[2][3] = -1
	m[3][2] = -2 * (f * n) / (f - n)
	m[3][0] = 0
	m[3][1] = 0
	m[3][3] = 0
}

func (m *Mat4x4) Ortho(l, r, b, t, n, f float32) {
	m[0][0] = 2 / (r - l)
	m[0][1] = 0
	m[0][2] = 0
	m[0][3] = 0
	m[1][1] = 2 / (t - b)
	m[1][0] = 0
	m[1][2] = 0
	m[1][3] = 0
	m[2][2] = -2 / (f - n)
	m[2][0] = 0
	m[2][1] = 0
	m[2][3] = 0
	m[3][0] = -(r + l) / (r - l)
	m[3][1] = -(t + b) / (t - b)
	m[3][2] = -(f + n) / (f - n)
	m[3][3] = 1
}

func (m *Mat4x4) Perspective(y_fov, aspect, n, f float32) {
	var a float32 = 1 / tanf(y_fov/2)
	m[0][0] = a / aspect
	m[0][1] = 0
	m[0][2] = 0
	m[0][3] = 0
	m[1][0] = 0
	m[1][1] = a
	m[1][2] = 0
	m[1][3] = 0
	m[2][0] = 0
	m[2][1] = 0
	m[2][2] = -((f + n) / (f - n))
	m[2][3] = -1
	m[3][0] = 0
	m[3][1] = 0
	m[3][2] = -((2 * f * n) / (f - n))
	m[3][3] = 0
}

func (m *Mat4x4) LookAt(eye, center, up *Vec3) {
	var f = new(Vec3)
	f.Sub(center, eye)
	f.Norm(f)
	var s = new(Vec3)
	s.MultCross(f, up)
	s.Norm(s)
	var t = new(Vec3)
	t.MultCross(s, f)
	m[0][0] = s[0]
	m[0][1] = t[0]
	m[0][2] = -f[0]
	m[0][3] = 0
	m[1][0] = s[1]
	m[1][1] = t[1]
	m[1][2] = -f[1]
	m[1][3] = 0
	m[2][0] = s[2]
	m[2][1] = t[2]
	m[2][2] = -f[2]
	m[2][3] = 0
	m[3][0] = 0
	m[3][1] = 0
	m[3][2] = 0
	m[3][3] = 1
	m.TranslateInPlace(-eye[0], -eye[1], -eye[2])
}

func (m *Mat4x4) FromQuat(q *Quat) {
	var a float32 = q[3]
	var b float32 = q[0]
	var c float32 = q[1]
	var d float32 = q[2]
	var a2 float32 = a * a
	var b2 float32 = b * b
	var c2 float32 = c * c
	var d2 float32 = d * d
	m[0][0] = a2 + b2 - c2 - d2
	m[0][1] = 2 * (b*c + a*d)
	m[0][2] = 2 * (b*d - a*c)
	m[0][3] = 0
	m[1][0] = 2 * (b*c - a*d)
	m[1][1] = a2 - b2 + c2 - d2
	m[1][2] = 2 * (c*d + a*b)
	m[1][3] = 0
	m[2][0] = 2 * (b*d + a*c)
	m[2][1] = 2 * (c*d - a*b)
	m[2][2] = a2 - b2 - c2 + d2
	m[2][3] = 0
	m[3][0] = 0
	m[3][1] = 0
	m[3][2] = 0
	m[3][3] = 1
}

func (r *Mat4x4) MultQuat(m *Mat4x4, q *Quat) {
	r[0].QuatMultVec4(q, &m[0])
	r[1].QuatMultVec4(q, &m[1])
	r[2].QuatMultVec4(q, &m[2])
	r[3][0] = 0
	r[3][1] = 0
	r[3][2] = 0
	r[3][3] = 1
}

func (m *Mat4x4) Data() []byte {
	const mm = 0x7fffffff
	return (*[mm]byte)(unsafe.Pointer(m))[:SizeofMat4x4]
}

// InvertMatrix computes the inverse of a 4x4 matrix using cofactor expansion.
// Returns the identity matrix if the input is singular.
func InvertMatrix(m *Mat4x4) Mat4x4 {
	var inv Mat4x4
	s := [6]float32{
		m[0][0]*m[1][1] - m[1][0]*m[0][1], m[0][0]*m[1][2] - m[1][0]*m[0][2], m[0][0]*m[1][3] - m[1][0]*m[0][3],
		m[0][1]*m[1][2] - m[1][1]*m[0][2], m[0][1]*m[1][3] - m[1][1]*m[0][3], m[0][2]*m[1][3] - m[1][2]*m[0][3],
	}
	c := [6]float32{
		m[2][0]*m[3][1] - m[3][0]*m[2][1], m[2][0]*m[3][2] - m[3][0]*m[2][2], m[2][0]*m[3][3] - m[3][0]*m[2][3],
		m[2][1]*m[3][2] - m[3][1]*m[2][2], m[2][1]*m[3][3] - m[3][1]*m[2][3], m[2][2]*m[3][3] - m[3][2]*m[2][3],
	}
	det := s[0]*c[5] - s[1]*c[4] + s[2]*c[3] + s[3]*c[2] - s[4]*c[1] + s[5]*c[0]
	if math.Abs(float64(det)) < 1e-10 {
		inv.Identity()
		return inv
	}
	d := 1.0 / det
	inv[0][0] = (m[1][1]*c[5] - m[1][2]*c[4] + m[1][3]*c[3]) * d
	inv[0][1] = (-m[0][1]*c[5] + m[0][2]*c[4] - m[0][3]*c[3]) * d
	inv[0][2] = (m[3][1]*s[5] - m[3][2]*s[4] + m[3][3]*s[3]) * d
	inv[0][3] = (-m[2][1]*s[5] + m[2][2]*s[4] - m[2][3]*s[3]) * d
	inv[1][0] = (-m[1][0]*c[5] + m[1][2]*c[2] - m[1][3]*c[1]) * d
	inv[1][1] = (m[0][0]*c[5] - m[0][2]*c[2] + m[0][3]*c[1]) * d
	inv[1][2] = (-m[3][0]*s[5] + m[3][2]*s[2] - m[3][3]*s[1]) * d
	inv[1][3] = (m[2][0]*s[5] - m[2][2]*s[2] + m[2][3]*s[1]) * d
	inv[2][0] = (m[1][0]*c[4] - m[1][1]*c[2] + m[1][3]*c[0]) * d
	inv[2][1] = (-m[0][0]*c[4] + m[0][1]*c[2] - m[0][3]*c[0]) * d
	inv[2][2] = (m[3][0]*s[4] - m[3][1]*s[2] + m[3][3]*s[0]) * d
	inv[2][3] = (-m[2][0]*s[4] + m[2][1]*s[2] - m[2][3]*s[0]) * d
	inv[3][0] = (-m[1][0]*c[3] + m[1][1]*c[1] - m[1][2]*c[0]) * d
	inv[3][1] = (m[0][0]*c[3] - m[0][1]*c[1] + m[0][2]*c[0]) * d
	inv[3][2] = (-m[3][0]*s[3] + m[3][1]*s[1] - m[3][2]*s[0]) * d
	inv[3][3] = (m[2][0]*s[3] - m[2][1]*s[1] + m[2][2]*s[0]) * d
	return inv
}

// --- Utility types and constants ---

const (
	SizeofMat4x4 = int(unsafe.Sizeof(Mat4x4{}))
	SizeofVec4   = int(unsafe.Sizeof(Vec4{}))
	SizeofVec3   = int(unsafe.Sizeof(Vec3{}))
	SizeofVec2   = int(unsafe.Sizeof(Vec2{}))
)

func DumpMatrix(m *Mat4x4, note string) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "[mat4x4] %s: \n", note)
	for i := 0; i < 4; i++ {
		fmt.Fprintf(buf, "%.3f, %.3f, %.3f, %.3f\n", m[i][0], m[i][1], m[i][2], m[i][3])
	}
	return buf.String()
}

type ArrayFloat32 []float32

func (a ArrayFloat32) Sizeof() int {
	return len(a) * 4
}

func (a ArrayFloat32) Data() []byte {
	const m = 0x7fffffff
	return (*[m]byte)(unsafe.Pointer((*mathSliceHeader)(unsafe.Pointer(&a)).Data))[:len(a)*4]
}

type mathSliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
