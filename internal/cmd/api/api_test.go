package api

import (
	"errors"
	"testing"
)

func TestDetectOperationKind(t *testing.T) {
	tests := []struct {
		in   string
		want operationKind
	}{
		{"{ viewer { id } }", operationKindQuery},
		{"query { viewer { id } }", operationKindQuery},
		{"mutation { deleteStack(id: \"x\") { id } }", operationKindMutation},
		{"subscription { runs { id } }", operationKindSubscription},
		{"stacks { id name }", operationKindSelectionSet},
		{"mutation2Stack { id }", operationKindSelectionSet},
		{"  # leading comment\nmutation { deleteStack(id: \"x\") { id } }", operationKindMutation},
		{"fragment StackFields on Stack { id }", operationKindUnknown},
	}
	for _, tc := range tests {
		if got := detectOperationKind(tc.in); got != tc.want {
			t.Errorf("detectOperationKind(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeDocument(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		allowMutation bool
		want          string
		wantErr       error
	}{
		{
			name:    "query shorthand braces stay unchanged",
			in:      "{ viewer { id } }",
			want:    "{ viewer { id } }",
			wantErr: nil,
		},
		{
			name:    "query selection set shorthand",
			in:      "stacks { id name }",
			want:    "query { stacks { id name } }",
			wantErr: nil,
		},
		{
			name:    "full query syntax",
			in:      "query { viewer { id } }",
			want:    "query { viewer { id } }",
			wantErr: nil,
		},
		{
			name:          "mutation selection set shorthand",
			in:            "stackDelete(id: \"x\") { id }",
			allowMutation: true,
			want:          "mutation { stackDelete(id: \"x\") { id } }",
			wantErr:       nil,
		},
		{
			name:    "field names containing keyword prefixes stay query shorthand",
			in:      "mutation2Stack { id }",
			want:    "query { mutation2Stack { id } }",
			wantErr: nil,
		},
		{
			name:          "full mutation syntax",
			in:            "mutation DeleteStack($id: ID!) { stackDelete(id: $id) { id } }",
			allowMutation: true,
			want:          "mutation DeleteStack($id: ID!) { stackDelete(id: $id) { id } }",
			wantErr:       nil,
		},
		{
			name:    "mutation requires flag",
			in:      "mutation { stackDelete(id: \"x\") { id } }",
			wantErr: errMutationFlagRequired,
		},
		{
			name:          "query rejected in mutation mode",
			in:            "query { viewer { id } }",
			allowMutation: true,
			wantErr:       errors.New("--mutation requires a GraphQL mutation document"),
		},
		{
			name:          "query shorthand rejected in mutation mode",
			in:            "{ viewer { id } }",
			allowMutation: true,
			wantErr:       errors.New("--mutation requires a GraphQL mutation document"),
		},
		{
			name:    "subscriptions rejected",
			in:      "subscription { runs { id } }",
			wantErr: errSubscriptionsNotAllowed,
		},
		{
			name:    "fragment must not be auto-wrapped",
			in:      "fragment StackFields on Stack { id }",
			wantErr: errors.New("could not determine GraphQL operation type; start the document with query or provide a bare query selection set"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeDocument(tc.in, tc.allowMutation)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr.Error() {
					t.Fatalf("got error %q, want %q", err, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseVariables(t *testing.T) {
	if v, err := parseVariables(""); v != nil || err != nil {
		t.Errorf("empty: got %v, %v", v, err)
	}
	if _, err := parseVariables("not-json"); err == nil {
		t.Error("expected error for invalid json")
	}
	if _, err := parseVariables("[1,2]"); err == nil {
		t.Error("expected error for non-object json")
	}
	v, err := parseVariables(`{"id":"my-stack"}`)
	if err != nil {
		t.Fatal(err)
	}
	if v["id"] != "my-stack" {
		t.Errorf("got %v", v)
	}
}

func TestGraphqlErrors(t *testing.T) {
	if msg := graphqlErrors([]byte(`{"data":{}}`)); msg != "" {
		t.Errorf("expected empty, got %q", msg)
	}
	if msg := graphqlErrors([]byte(`{"errors":[{"message":"bad"}]}`)); msg != "bad" {
		t.Errorf("got %q", msg)
	}
	if msg := graphqlErrors([]byte(`{"errors":[{"message":"a"},{"message":"b"}]}`)); msg != "a; b" {
		t.Errorf("got %q", msg)
	}
}

func TestTruncate(t *testing.T) {
	if s := truncate([]byte("short"), 100); s != "short" {
		t.Errorf("got %q", s)
	}
	if s := truncate([]byte("abcdef"), 3); s != "abc..." {
		t.Errorf("got %q", s)
	}
}
