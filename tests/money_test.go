package money_test

import (
	"testing"
	"webtyp.com/money"
)

func TestMoneyNew(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		c       money.Currency
		wantErr error
	}{
		{"valid", 100, "CLP", nil},
		{"invalid currency empty", 100, "", money.ErrCurrency},
		{"invalid currency lower", 100, "clp", money.ErrCurrency},
		{"invalid currency long", 100, "CLPP", money.ErrCurrency},
		{"invalid currency short", 100, "CL", money.ErrCurrency},
		{"invalid currency number", 100, "CL1", money.ErrCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := money.New(tt.amount, tt.c)
			if err != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if m.Amount != tt.amount {
					t.Errorf("New() amount = %v, want %v", m.Amount, tt.amount)
				}
				if m.Currency != tt.c {
					t.Errorf("New() currency = %v, want %v", m.Currency, tt.c)
				}
			}
		})
	}
}

func TestMoneyNeg(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		want    int64
		wantErr error
	}{
		{"positive", 100, -100, nil},
		{"negative", -100, 100, nil},
		{"zero", 0, 0, nil},
		{"min int64", -9223372036854775808, 0, money.ErrOverflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := money.Money{Amount: tt.amount, Currency: "CLP"}
			got, err := m.Neg()
			if err != tt.wantErr {
				t.Errorf("Neg() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got.Amount != tt.want {
				t.Errorf("Neg() = %v, want %v", got.Amount, tt.want)
			}
		})
	}
}

func TestMoneyAddSub(t *testing.T) {
	tests := []struct {
		name    string
		a       money.Money
		b       money.Money
		wantAdd int64
		errAdd  error
		wantSub int64
		errSub  error
	}{
		{
			name:    "same currency",
			a:       money.Money{Amount: 100, Currency: "CLP"},
			b:       money.Money{Amount: 50, Currency: "CLP"},
			wantAdd: 150, errAdd: nil,
			wantSub: 50, errSub: nil,
		},
		{
			name:    "mismatch currency",
			a:       money.Money{Amount: 100, Currency: "CLP"},
			b:       money.Money{Amount: 50, Currency: "USD"},
			wantAdd: 0, errAdd: money.ErrMismatch,
			wantSub: 0, errSub: money.ErrMismatch,
		},
		{
			name:    "overflow add",
			a:       money.Money{Amount: 9223372036854775807, Currency: "CLP"},
			b:       money.Money{Amount: 1, Currency: "CLP"},
			wantAdd: 0, errAdd: money.ErrOverflow,
			wantSub: 9223372036854775806, errSub: nil,
		},
		{
			name:    "overflow sub",
			a:       money.Money{Amount: -9223372036854775808, Currency: "CLP"},
			b:       money.Money{Amount: 1, Currency: "CLP"},
			wantAdd: -9223372036854775807, errAdd: nil,
			wantSub: 0, errSub: money.ErrOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdd, errAdd := money.Add(tt.a, tt.b)
			if errAdd != tt.errAdd {
				t.Errorf("Add() error = %v, wantErr %v", errAdd, tt.errAdd)
			}
			if errAdd == nil && gotAdd.Amount != tt.wantAdd {
				t.Errorf("Add() = %v, want %v", gotAdd.Amount, tt.wantAdd)
			}

			gotSub, errSub := money.Sub(tt.a, tt.b)
			if errSub != tt.errSub {
				t.Errorf("Sub() error = %v, wantErr %v", errSub, tt.errSub)
			}
			if errSub == nil && gotSub.Amount != tt.wantSub {
				t.Errorf("Sub() = %v, want %v", gotSub.Amount, tt.wantSub)
			}
		})
	}
}

func TestMoneyCompare(t *testing.T) {
	tests := []struct {
		name    string
		a       money.Money
		b       money.Money
		want    int
		wantErr error
	}{
		{
			name: "equal",
			a:    money.Money{Amount: 100, Currency: "CLP"},
			b:    money.Money{Amount: 100, Currency: "CLP"},
			want: 0, wantErr: nil,
		},
		{
			name: "less",
			a:    money.Money{Amount: 50, Currency: "CLP"},
			b:    money.Money{Amount: 100, Currency: "CLP"},
			want: -1, wantErr: nil,
		},
		{
			name: "greater",
			a:    money.Money{Amount: 150, Currency: "CLP"},
			b:    money.Money{Amount: 100, Currency: "CLP"},
			want: 1, wantErr: nil,
		},
		{
			name: "mismatch",
			a:    money.Money{Amount: 100, Currency: "CLP"},
			b:    money.Money{Amount: 100, Currency: "USD"},
			want: 0, wantErr: money.ErrMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := money.Compare(tt.a, tt.b)
			if err != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMoneyShare(t *testing.T) {
	tests := []struct {
		name        string
		amount      int64
		basisPoints int64
		want        int64
		wantErr     error
	}{
		{"example 1", 15001, 5000, 7501, nil},
		{"example 2", -15001, 5000, -7501, nil},
		{"example 3", 10000, 3333, 3333, nil},
		{"0 bp", 10000, 0, 0, nil},
		{"10000 bp", 10000, 10000, 10000, nil},
		{"10001 bp", 10000, 10001, 0, money.ErrBasisPoints},
		{"-1 bp", 10000, -1, 0, money.ErrBasisPoints},
		{"overflow multiply", 9223372036854775807, 5000, 0, money.ErrOverflow},
		{"min int64 zero share", -9223372036854775808, 0, 0, nil},
		{"min int64 non-zero share", -9223372036854775808, 5000, 0, money.ErrOverflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := money.Money{Amount: tt.amount, Currency: "CLP"}
			got, err := money.Share(m, tt.basisPoints)
			if err != tt.wantErr {
				t.Errorf("Share() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got.Amount != tt.want {
				t.Errorf("Share() = %v, want %v", got.Amount, tt.want)
			}
		})
	}
}
