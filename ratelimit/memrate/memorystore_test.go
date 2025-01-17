package memrate

import (
	"context"
	lru "github.com/hashicorp/golang-lru"
	"github.com/unionj-cloud/go-doudou/ratelimit"
	"reflect"
	"testing"
)

func TestMemoryStore_addKey(t *testing.T) {
	type fields struct {
		keys      map[string]ratelimit.Limiter
		limiterFn LimiterFn
	}
	type args struct {
		key string
	}
	limiter := NewLimiter(1, 3)
	tests := []struct {
		name   string
		fields fields
		args   args
		want   ratelimit.Limiter
	}{
		{
			name: "",
			fields: fields{
				keys: make(map[string]ratelimit.Limiter),
				limiterFn: func(ctx context.Context, store *MemoryStore, key string) ratelimit.Limiter {
					return NewLimiter(1, 3)
				},
			},
			args: args{
				key: "192.168.1.6:8080",
			},
			want: limiter,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, _ := lru.New(256)
			store := &MemoryStore{
				keys:      keys,
				limiterFn: tt.fields.limiterFn,
			}
			if got := store.addKeyCtx(context.Background(), tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryStore_GetLimiter(t *testing.T) {
	type fields struct {
		keys      map[string]ratelimit.Limiter
		limiterFn LimiterFn
	}
	type args struct {
		key string
	}
	limiter := NewLimiter(1, 3)
	tests := []struct {
		name   string
		fields fields
		args   args
		want   ratelimit.Limiter
	}{
		{
			name: "",
			fields: fields{
				keys: make(map[string]ratelimit.Limiter),
				limiterFn: func(ctx context.Context, store *MemoryStore, key string) ratelimit.Limiter {
					return NewLimiter(1, 3)
				},
			},
			args: args{
				key: "192.168.1.6:8080",
			},
			want: limiter,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, _ := lru.New(256)
			store := &MemoryStore{
				keys:      keys,
				limiterFn: tt.fields.limiterFn,
			}
			if got := store.GetLimiter(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLimiter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryStore_DeleteKey(t *testing.T) {
	type fields struct {
		keys      map[string]ratelimit.Limiter
		limiterFn LimiterFn
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "",
			fields: fields{
				keys: make(map[string]ratelimit.Limiter),
				limiterFn: func(ctx context.Context, store *MemoryStore, key string) ratelimit.Limiter {
					return NewLimiter(1, 3)
				},
			},
			args: args{
				key: "192.168.1.6:8080",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, _ := lru.New(256)
			store := &MemoryStore{
				keys:      keys,
				limiterFn: tt.fields.limiterFn,
			}
			store.addKeyCtx(context.Background(), tt.args.key)
			if exists := store.keys.Contains(tt.args.key); !exists {
				t.Error("key should exists")
			}
			store.DeleteKey(tt.args.key)
			if exists := store.keys.Contains(tt.args.key); exists {
				t.Error("key should not exists")
			}
		})
	}
}

func TestNewMemoryStore(t *testing.T) {
	type args struct {
		fn LimiterFn
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "",
			args: args{
				fn: func(ctx context.Context, store *MemoryStore, key string) ratelimit.Limiter {
					return NewLimiter(1, 3)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMemoryStore(tt.args.fn); got == nil {
				t.Error("NewMemoryStore() = nil")
			}
		})
	}
}
