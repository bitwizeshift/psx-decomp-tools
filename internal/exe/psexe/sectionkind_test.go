package psexe_test

import (
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/exe/psexe"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSectionKindString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		kind psexe.SectionKind
		want string
	}{
		{
			name: "Text",
			kind: psexe.Text,
			want: "text",
		}, {
			name: "Data",
			kind: psexe.Data,
			want: "data",
		}, {
			name: "BSS",
			kind: psexe.BSS,
			want: "bss",
		}, {
			name: "Stack",
			kind: psexe.Stack,
			want: "stack",
		}, {
			name: "Unknown",
			kind: psexe.SectionKind(99),
			want: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			kind := tc.kind

			// Act
			name := kind.String()

			// Assert
			if got, want := name, tc.want; !cmp.Equal(got, want) {
				t.Errorf("String() = %q, want %q", got, want)
			}
		})
	}
}

func TestSectionKindMarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		kind    psexe.SectionKind
		want    string
		wantErr error
	}{
		{
			name: "Text",
			kind: psexe.Text,
			want: "text",
		}, {
			name: "Stack",
			kind: psexe.Stack,
			want: "stack",
		}, {
			name:    "Unknown",
			kind:    psexe.SectionKind(99),
			wantErr: psexe.ErrBadSection,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			kind := tc.kind

			// Act
			text, err := kind.MarshalText()

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("MarshalText() error = %v, want %v", got, want)
			}
			if got, want := string(text), tc.want; !cmp.Equal(got, want) {
				t.Errorf("MarshalText() = %q, want %q", got, want)
			}
		})
	}
}

func TestSectionKindUnmarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		want    psexe.SectionKind
		wantErr error
	}{
		{
			name: "Text",
			text: "text",
			want: psexe.Text,
		}, {
			name: "BSS",
			text: "bss",
			want: psexe.BSS,
		}, {
			name:    "Unknown",
			text:    "heap",
			wantErr: psexe.ErrBadSection,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var kind psexe.SectionKind

			// Act
			err := kind.UnmarshalText([]byte(tc.text))

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("UnmarshalText() error = %v, want %v", got, want)
			}
			if got, want := kind, tc.want; !cmp.Equal(got, want) {
				t.Errorf("UnmarshalText() kind = %v, want %v", got, want)
			}
		})
	}
}
