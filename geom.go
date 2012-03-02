// Copyright 2012 Daniel Connelly.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtreego

import (
	"fmt"
	"math"
	"strings"
)

// DimError represents a failure due to mismatched dimensions.
type DimError struct {
	Expected int
	Actual   int
}

func (err *DimError) Error() string {
	return "rtreego: dimension mismatch"
}

// DistError is an improper distance measurement.  It implements the error
// and is generated when a distance-related assertion fails.
type DistError float64

func (err DistError) Error() string {
	return "rtreego: improper distance"
}

// Point represents a point in n-dimensional Euclidean space.
type Point []float64

// Dist computes the Euclidean distance between two points p and q.
func Dist(p, q Point) (float64, error) {
	if len(p) != len(q) {
		return 0, &DimError{len(p), len(q)}
	}
	sum := 0.0
	for i := range p {
		dx := p[i] - q[i]
		sum += dx * dx
	}
	return math.Sqrt(sum), nil
}

// MinDist computes the square of the distance from a point to a rectangle.
// If the point is contained in the rectangle then the distance is zero.
// Implemented per Definition 2 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func MinDist(p Point, r *Rect) (float64, error) {
	if len(p) != len(r.p) {
		return 0, &DimError{len(p), len(r.p)}
	}

	sum := 0.0
	for i, pi := range p {
		if pi < r.p[i] {
			d := pi - r.p[i]
			sum += d * d
		} else if pi > r.q[i] {
			d := pi - r.q[i]
			sum += d * d
		} else {
			sum += 0
		}
	}
	return sum, nil
}

// MinMaxDist computes the minimum of the maximum distances from p to points
// on r.  If r is the bounding box of some geometric objects, then there is
// at least one object contained in r within MinMaxDist(p, r) of p.
// Implemented per Definition 4 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func MinMaxDist(p Point, r *Rect) (float64, error) {
	if len(p) != len(r.p) {
		return 0, &DimError{len(p), len(r.p)}
	}

	// by definition, MinMaxDist(p, r) =
	// min{1<=k<=n}(|pk - rmk|^2 + sum{1<=i<=n, i != k}(|pi - rMi|^2))
	// where rmk and rMk are defined as follows:

	rm := func (k int) float64 {
		if p[k] <= (r.p[k] + r.q[k]) / 2 {
			return r.p[k]
		}
		return r.q[k]
	}

	rM := func (k int) float64 {
		if p[k] >= (r.p[k] + r.q[k]) / 2 {
			return r.p[k]
		}
		return r.q[k]
	}

	// This formula can be computed in linear time by precomputing
	// S = sum{1<=i<=n}(|pi - rMi|^2).

	S := 0.0
	for i := range p {
		d := p[i] - rM(i)
		S += d*d
	}

	// Compute MinMaxDist using the precomputed S.
	min := math.MaxFloat64
	for k := range p {
		d1 := p[k] - rM(k)
		d2 := p[k] - rm(k)
		d := S - d1*d1 + d2*d2
		if d < min {
			min = d
		}
	}

	return min, nil
}

// Rect represents a subset of n-dimensional Euclidean space of the form
// [a1, b1] x [a2, b2] x ... x [an, bn], where ai < bi for all 1 <= i <= n.
type Rect struct {
	p, q Point // Enforced by NewRect: p[i] <= q[i] for all i.
}

func (r *Rect) String() string {
	s := make([]string, len(r.p))
	for i, a := range r.p {
		b := r.q[i]
		s[i] = fmt.Sprintf("[%.2f, %.2f]", a, b)
	}
	return strings.Join(s, "x")
}

// NewRect constructs and returns a pointer to a Rect given a corner point and
// the lengths of each dimension.  The point p should be the most-negative point
// on the rectangle (in every dimension) and every length should be positive.
func NewRect(p Point, lengths []float64) (*Rect, error) {
	if len(p) != len(lengths) {
		return nil, &DimError{len(p), len(lengths)}
	}
	q := make([]float64, len(p))
	for i := range p {
		if lengths[i] <= 0 {
			return nil, DistError(lengths[i])
		}
		q[i] = p[i] + lengths[i]
	}
	return &Rect{p, q}, nil
}

// Size computes the measure of a rectangle (the product of its side lengths).
func (r *Rect) Size() float64 {
	size := 1.0
	for i, a := range r.p {
		b := r.q[i]
		size *= b - a
	}
	return size
}

// Margin computes the sum of the edge lengths of a rectangle.
func (r *Rect) Margin() float64 {
	// The number of edges in an n-dimensional rectangle is n * 2^(n-1)
	// (http://en.wikipedia.org/wiki/Hypercube_graph).  Thus the number
	// of edges of length (ai - bi), where the rectangle is determined
	// by p = (a1, a2, ..., an) and q = (b1, b2, ..., bn), is 2^(n-1).
	// The margin of the rectangle, then, is given by the formula
	// 2^(n-1) * [(b1 - a1) + (b2 - a2) + ... + (bn - an)].
	dim := len(r.p)
	sum := 0.0
	for i, a := range r.p {
		b := r.q[i]
		sum += b - a
	}
	return math.Pow(2, float64(dim-1)) * sum
}

// ContainsPoint tests whether p is located inside or on the boundary of r.
func (r *Rect) ContainsPoint(p Point) (bool, error) {
	if len(p) != len(r.p) {
		return false, &DimError{len(r.p), len(p)}
	}

	for i, a := range p {
		// p is contained in (or on) r if and only if p <= a <= q for
		// every dimension.
		if a < r.p[i] || a > r.q[i] {
			return false, nil
		}
	}

	return true, nil
}

// ContainsRect tests whether r2 is is located inside r1.
func (r1 *Rect) ContainsRect(r2 *Rect) (bool, error) {
	if len(r1.p) != len(r2.p) {
		return false, &DimError{len(r1.p), len(r2.p)}
	}

	for i, a1 := range r1.p {
		b1, a2, b2 := r1.q[i], r2.p[i], r2.q[i]
		// enforced by constructor: a1 <= b1 and a2 <= b2.
		// so containment holds if and only if a1 <= a2 <= b2 <= b1
		// for every dimension.
		if a1 > a2 || b2 > b1 {
			return false, nil
		}
	}

	return true, nil
}

// Intersect computes the intersection of two rectangles.  If no intersection
// exists, the intersection is nil.
func Intersect(r1, r2 *Rect) (*Rect, error) {
	dim := len(r1.p)
	if len(r2.p) != dim {
		return nil, &DimError{dim, len(r2.p)}
	}

	// There are four cases of overlap:
	//
	// 1.  a1------------b1
	//          a2------------b2
	//          p--------q
	//
	// 2.       a1------------b1
	//     a2------------b2
	//          p--------q
	//
	// 3.  a1-----------------b1
	//          a2-------b2
	//          p--------q
	//
	// 4.       a1-------b1
	//     a2-----------------b2
	//          p--------q
	//
	// Thus there are only two cases of non-overlap:
	//
	// 1. a1------b1
	//                a2------b2
	//
	// 2.             a1------b1
	//    a2------b2
	//
	// Enforced by constructor: a1 <= b1 and a2 <= b2.  So we can just
	// check the endpoints.

	p := make([]float64, dim)
	q := make([]float64, dim)
	for i := range p {
		a1, b1, a2, b2 := r1.p[i], r1.q[i], r2.p[i], r2.q[i]
		if b2 <= a1 || b1 <= a2 {
			return nil, nil
		}
		p[i] = math.Max(a1, a2)
		q[i] = math.Min(b1, b2)
	}
	return &Rect{p, q}, nil
}

// ToRect constructs a rectangle containing p with side lengths 2*tol.
func (p Point) ToRect(tol float64) *Rect {
	dim := len(p)
	a, b := make([]float64, dim), make([]float64, dim)
	for i := range p {
		a[i] = p[i] - tol
		b[i] = p[i] + tol
	}
	return &Rect{a, b}
}

// BoundingBox constructs the smallest rectangle containing both r1 and r2.
func BoundingBox(r1, r2 *Rect) (*Rect, error) {
	dim := len(r1.p)
	if len(r2.p) != dim {
		return nil, &DimError{dim, len(r2.p)}
	}

	p := make([]float64, dim)
	lengths := make([]float64, dim)
	for i := range p {
		if r1.p[i] <= r2.p[i] {
			p[i] = r1.p[i]
		} else {
			p[i] = r2.p[i]
		}

		if r1.q[i] <= r2.q[i] {
			lengths[i] = r2.q[i] - p[i]
		} else {
			lengths[i] = r1.q[i] - p[i]
		}
	}

	return NewRect(p, lengths)
}
