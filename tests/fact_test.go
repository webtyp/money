package money_test

import (
	"testing"
	"webtyp.com/money"
)

func TestFactValidate(t *testing.T) {
	validAmount, _ := money.New(100, "CLP")
	invalidAmount, _ := money.New(0, "CLP")
	validSource := money.Ref{Type: "payment", ID: "123"}
	emptySource := money.Ref{}

	tests := []struct {
		name    string
		f       money.Fact
		wantErr error
	}{
		{
			name: "valid CHARGED",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: nil,
		},
		{
			name: "valid REFUNDED",
			f: money.Fact{
				Kind:   money.Refunded,
				Side:   money.Outflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: nil,
		},
		{
			name: "valid ACCRUED",
			f: money.Fact{
				Kind:   money.Accrued,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
				DueAt:  2000,
			},
			wantErr: nil,
		},
		{
			name: "valid SETTLED",
			f: money.Fact{
				Kind:   money.Settled,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: nil,
		},
		{
			name: "valid REVERSED",
			f: money.Fact{
				Kind:   money.Reversed,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
				Of:     validSource,
			},
			wantErr: nil,
		},
		{
			name: "valid PURCHASED",
			f: money.Fact{
				Kind:   money.Purchased,
				Side:   money.Outflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: nil,
		},
		{
			name: "valid CONSUMED",
			f: money.Fact{
				Kind:   money.Consumed,
				Side:   money.Outflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: nil,
		},
		{
			name: "invalid kind",
			f: money.Fact{
				Kind:   "INVALID",
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrKind,
		},
		{
			name: "invalid side general",
			f: money.Fact{
				Kind:   money.Accrued,
				Side:   0,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrSide,
		},
		{
			name: "invalid side for CHARGED",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Outflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrSide,
		},
		{
			name: "invalid side for REFUNDED",
			f: money.Fact{
				Kind:   money.Refunded,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrSide,
		},
		{
			name: "invalid amount (zero)",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: invalidAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrAmount,
		},
		{
			name: "invalid currency",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: money.Money{Amount: 100, Currency: "clp"},
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrCurrency,
		},
		{
			name: "missing source",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: emptySource,
				At:     1000,
			},
			wantErr: money.ErrSource,
		},
		{
			name: "missing scope",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrScope,
		},
		{
			name: "missing time",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     0,
			},
			wantErr: money.ErrTime,
		},
		{
			name: "due time on non-accrued",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
				DueAt:  2000,
			},
			wantErr: money.ErrDueTime,
		},
		{
			name: "negative due time on accrued",
			f: money.Fact{
				Kind:   money.Accrued,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
				DueAt:  -1,
			},
			wantErr: money.ErrDueTime,
		},
		{
			name: "missing Of on reversed",
			f: money.Fact{
				Kind:   money.Reversed,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
			},
			wantErr: money.ErrReverses,
		},
		{
			name: "Of set on non-reversed",
			f: money.Fact{
				Kind:   money.Charged,
				Side:   money.Inflow,
				Amount: validAmount,
				Scope:  "test",
				Source: validSource,
				At:     1000,
				Of:     validSource,
			},
			wantErr: money.ErrReverses,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Validate(); err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFactKey(t *testing.T) {
	f := money.Fact{
		Kind: money.Charged,
		Source: money.Ref{
			Type: "payment",
			ID:   "123",
		},
	}
	want := "CHARGED|payment|123"
	if got := f.Key(); got != want {
		t.Errorf("Key() = %v, want %v", got, want)
	}
}
