package fest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMatchPessimistic_SingleComponent tests pessimistic version matching with single component.
// Bug: matchPessimistic fails when want has fewer parts than have (e.g., want="5", have="5.1.0").
// Ruby's ~>5 means ">= 5.0, < 6.0", but the code checks len(haveParts) < len(wantParts).
func TestMatchPessimistic_SingleComponent(t *testing.T) {
	tests := []struct {
		name string
		want string
		have string
		exp  bool
	}{
		{
			name: "major only constraint matches minor version",
			want: "puma@~>5",
			have: "puma@5.1.0",
			exp:  true, // ~>5 means >= 5.0, < 6.0
		},
		{
			name: "major only constraint matches patch version",
			want: "rails@~>7",
			have: "rails@7.0.8",
			exp:  true,
		},
		{
			name: "major only constraint rejects wrong major",
			want: "puma@~>5",
			have: "puma@6.0.0",
			exp:  false,
		},
		{
			name: "major only constraint accepts exact major",
			want: "puma@~>5",
			have: "puma@5.0.0",
			exp:  true,
		},
	}

	r := rubyGems{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := r.Match(tc.want, tc.have)
			require.Equal(t, tc.exp, result,
				"Match(%q, %q) = %v, want %v", tc.want, tc.have, result, tc.exp)
		})
	}
}

// TestMatchPessimistic_BoundaryCheck tests pessimistic version upper boundary.
// Bug: Pessimistic match doesn't enforce upper bound correctly.
func TestMatchPessimistic_BoundaryCheck(t *testing.T) {
	tests := []struct {
		name string
		want string
		have string
		exp  bool
	}{
		{
			name: "minor constraint allows patch bumps",
			want: "gem@~>1.2",
			have: "gem@1.2.9",
			exp:  true,
		},
	{
		name: "minor constraint accepts minor bump within major",
		want: "gem@~>1.2",
		have: "gem@1.3.0",
		exp:  true, // ~>1.2 means >= 1.2, < 2.0
	},
		{
			name: "patch constraint allows patch bumps",
			want: "gem@~>1.2.3",
			have: "gem@1.2.5",
			exp:  true,
		},
		{
			name: "patch constraint rejects minor bump",
			want: "gem@~>1.2.3",
			have: "gem@1.3.0",
			exp:  false,
		},
	}

	r := rubyGems{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := r.Match(tc.want, tc.have)
			require.Equal(t, tc.exp, result,
				"Match(%q, %q) = %v, want %v", tc.want, tc.have, result, tc.exp)
		})
	}
}

// TestMatch_GreaterThanOrEqual tests >= version constraints.
func TestMatch_GreaterThanOrEqual(t *testing.T) {
	tests := []struct {
		name string
		want string
		have string
		exp  bool
	}{
		{
			name: "greater than minimum",
			want: "gem@>=1.0.0",
			have: "gem@1.5.0",
			exp:  true,
		},
		{
			name: "equal to minimum",
			want: "gem@>=1.0.0",
			have: "gem@1.0.0",
			exp:  true,
		},
		{
			name: "less than minimum",
			want: "gem@>=1.0.0",
			have: "gem@0.9.0",
			exp:  false,
		},
	}

	r := rubyGems{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := r.Match(tc.want, tc.have)
			require.Equal(t, tc.exp, result,
				"Match(%q, %q) = %v, want %v", tc.want, tc.have, result, tc.exp)
		})
	}
}

// TestMatch_ExactVersion tests exact version matching.
func TestMatch_ExactVersion(t *testing.T) {
	tests := []struct {
		name string
		want string
		have string
		exp  bool
	}{
		{
			name: "exact match",
			want: "gem@1.2.3",
			have: "gem@1.2.3",
			exp:  true,
		},
		{
			name: "version mismatch",
			want: "gem@1.2.3",
			have: "gem@1.2.4",
			exp:  false,
		},
		{
			name: "equals prefix match",
			want: "gem@=1.2.3",
			have: "gem@1.2.3",
			exp:  true,
		},
	}

	r := rubyGems{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := r.Match(tc.want, tc.have)
			require.Equal(t, tc.exp, result,
				"Match(%q, %q) = %v, want %v", tc.want, tc.have, result, tc.exp)
		})
	}
}

// TestCompareVer_EdgeCases tests version comparison edge cases.
func TestCompareVer_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		exp  int
	}{
		{
			name: "valid versions equal",
			v1:   "1.0.0",
			v2:   "1.0.0",
			exp:  0,
		},
		{
			name: "valid v1 greater",
			v1:   "2.0.0",
			v2:   "1.0.0",
			exp:  1,
		},
		{
			name: "valid v1 less",
			v1:   "1.0.0",
			v2:   "2.0.0",
			exp:  -1,
		},
		{
			name: "invalid versions fall back to string compare",
			v1:   "abc",
			v2:   "def",
			exp:  -1,
		},
	}

	r := rubyGems{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := r.compareVer(tc.v1, tc.v2)
			require.Equal(t, tc.exp, result,
				"compareVer(%q, %q) = %v, want %v", tc.v1, tc.v2, result, tc.exp)
		})
	}
}
