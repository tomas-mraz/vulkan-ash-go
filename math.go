package asch

import (
	"math"

	lin "github.com/xlab/linmath"
)

// InvertMatrix computes the inverse of a 4x4 matrix using cofactor expansion.
// Returns the identity matrix if the input is singular.
func InvertMatrix(m *lin.Mat4x4) lin.Mat4x4 {
	var inv lin.Mat4x4
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

// AlignUp rounds size up to the nearest multiple of alignment.
func AlignUp(size, alignment uint32) uint32 {
	return (size + alignment - 1) &^ (alignment - 1)
}
