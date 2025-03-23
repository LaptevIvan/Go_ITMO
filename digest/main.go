package go_digest

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
	"strings"
	"unsafe"
)

// GetCharByIndex returns the i-th character from the given string.
func GetCharByIndex(str string, idx int) rune {
	i := 0
	for _, r := range str {
		if i < idx {
			i++
			continue
		}
		return r
	}
	panic("idx out of bounds slice")
}

// GetStringBySliceOfIndexes returns a string formed by concatenating specific characters from the input string based
// on the provided indexes.
func GetStringBySliceOfIndexes(str string, indexes []int) string {
	builder := &strings.Builder{}
	builder.Grow(len(indexes) * 4)
	strRune := []rune(str)
	for _, ind := range indexes {
		builder.WriteRune(strRune[ind])
	}
	return builder.String()
}

// ShiftPointer shifts the given pointer by the specified number of bytes using unsafe.Add.
func ShiftPointer(pointer **int, shift int) {
	*pointer = (*int)(unsafe.Add(unsafe.Pointer(*pointer), shift))
}

const eps = 1.e-6

var cmp = func(a, b float64) bool {
	return a == b || math.Abs(a-b) <= eps
}

// IsComplexEqual compares two complex numbers and determines if they are equal.
func IsComplexEqual(a, b complex128) bool {
	return cmp(real(a), real(b)) && cmp(imag(a), imag(b))
}

// GetRootsOfQuadraticEquation returns two roots of a quadratic equation ax^2 + bx + c = 0.
func GetRootsOfQuadraticEquation(a, b, c float64) (complex128, complex128) {
	aC := complex(2*a, 0)
	bC := complex(-b, 0)
	sqrtD := cmplx.Sqrt(complex(b*b-4*a*c, 0))
	return (bC + sqrtD) / aC, (bC - sqrtD) / aC
}

// Sort sorts in-place the given slice of integers in ascending order.
func Sort(source []int) {
	var quicksort func(arr []int)
	quicksort = func(arr []int) {
		n := len(arr)
		if n <= 1 {
			return
		}
		ind := rand.IntN(n)
		el := arr[ind]
		less, eq := 0, 0
		for i, curEl := range arr {
			if curEl == el {
				arr[eq], arr[i] = curEl, arr[eq]
				eq++
				continue
			}
			if curEl < el {
				arr[eq], arr[i] = curEl, arr[eq]
				arr[less], arr[eq] = arr[eq], arr[less]
				less++
				eq++
			}
		}
		quicksort(arr[0:less])
		quicksort(arr[eq:])
	}
	quicksort(source)
}

// ReverseSliceOne in-place reverses the order of elements in the given slice.
func ReverseSliceOne(s []int) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// ReverseSliceTwo returns a new slice of integers with elements in reverse order compared to the input slice.
// The original slice remains unmodified.
func ReverseSliceTwo(s []int) []int {
	ans := make([]int, len(s))
	copy(ans, s)
	ReverseSliceOne(ans)
	return ans
}

// SwapPointers swaps the values of two pointers.
func SwapPointers(a, b *int) {
	*a, *b = *b, *a
}

// IsSliceEqual compares two slices of integers and returns true if they contain the same elements in the same order.
func IsSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// DeleteByIndex deletes the element at the specified index from the slice and returns a new slice.
// The original slice remains unmodified.
func DeleteByIndex(s []int, idx int) []int {
	ans := make([]int, len(s)-1)
	copy(ans, s[0:idx])
	copy(ans[idx:], s[idx+1:])
	return ans
}
