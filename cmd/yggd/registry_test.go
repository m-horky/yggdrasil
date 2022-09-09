package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSet(t *testing.T) {
	tests := []struct {
		description string
		input       []struct {
			directive string
			worker    *workerConfig
		}
		want      *registry
		wantError error
	}{
		{
			description: "two directive values",
			input: []struct {
				directive string
				worker    *workerConfig
			}{
				{
					directive: "test1",
					worker: &workerConfig{
						directive: "test1",
					},
				},
				{
					directive: "test2",
					worker: &workerConfig{
						directive: "test2",
					},
				},
			},
			want: &registry{
				mp: map[string]*workerConfig{
					"test1": {
						directive: "test1",
					},
					"test2": {
						directive: "test2",
					},
				},
			},
		},
		{
			description: "duplicate directive value",
			input: []struct {
				directive string
				worker    *workerConfig
			}{
				{
					directive: "test",
					worker: &workerConfig{
						directive: "test",
					},
				},
				{
					directive: "test",
					worker: &workerConfig{
						directive: "test",
					},
				},
			},
			wantError: errWorkerRegistered,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got := &registry{}

			for _, input := range test.input {
				if err := got.set(input.directive, input.worker); err != nil {
					if err != nil {
						if test.wantError != nil {
							if !cmp.Equal(err, test.wantError) {
								t.Fatalf("%#v != %#v", err, test.wantError)
							}
						} else {
							t.Fatalf("unexpected error: %v", err)
						}
					}
				}
			}

			if test.wantError == nil {
				if !cmp.Equal(got, test.want, cmp.AllowUnexported(registry{}, workerConfig{}), cmpopts.IgnoreFields(registry{}, "mu")) {
					t.Fatalf("%#v != %#v", got, test.want)
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		description string
		input       struct {
			r         *registry
			directive string
		}
		want *workerConfig
	}{
		{
			description: "present",
			input: struct {
				r         *registry
				directive string
			}{
				r: &registry{
					mp: map[string]*workerConfig{
						"test": {
							directive: "test",
						},
					},
				},
				directive: "test",
			},
			want: &workerConfig{
				directive: "test",
			},
		},
		{
			description: "absent",
			input: struct {
				r         *registry
				directive string
			}{
				r: &registry{
					mp: map[string]*workerConfig{
						"test": {
							directive: "test",
						},
					},
				},
				directive: "test2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got := test.input.r.get(test.input.directive)

			if !cmp.Equal(got, test.want, cmp.AllowUnexported(registry{}, workerConfig{}), cmpopts.IgnoreFields(registry{}, "mu")) {
				t.Errorf("%#v != %#v", got, test.want)
			}
		})
	}
}

func TestDel(t *testing.T) {
	tests := []struct {
		description string
		input       struct {
			r *registry
			h string
		}
		want *registry
	}{
		{
			input: struct {
				r *registry
				h string
			}{
				r: &registry{
					mp: map[string]*workerConfig{
						"test": {
							directive: "test",
						},
					},
				},
				h: "test",
			},
			want: &registry{
				mp: map[string]*workerConfig{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got := &registry{
				mp: test.input.r.mp,
			}
			got.del(test.input.h)

			if !cmp.Equal(got, test.want, cmp.AllowUnexported(registry{}, workerConfig{}), cmpopts.IgnoreFields(registry{}, "mu")) {
				t.Errorf("%#v != %#v", got, test.want)
			}
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		description string
		input       *registry
		want        map[string]*workerConfig
	}{
		{
			input: &registry{
				mp: map[string]*workerConfig{
					"test1": {
						directive: "test1",
					},
					"test2": {
						directive: "test2",
					},
				},
			},
			want: map[string]*workerConfig{
				"test1": {
					directive: "test1",
				},
				"test2": {
					directive: "test2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got := test.input.all()

			if !cmp.Equal(got, test.want, cmp.AllowUnexported(registry{}, workerConfig{}), cmpopts.IgnoreFields(registry{}, "mu")) {
				t.Errorf("%#v != %#v", got, test.want)
			}
		})
	}
}
