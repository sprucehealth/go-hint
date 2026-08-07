package hint

import "testing"

func TestOperationEncode(t *testing.T) {
	cases := []struct {
		name      string
		operation *Operation
		expected  string
	}{
		{"greater than", &Operation{Operator: OperatorGreaterThan, Operand: "1"}, `"gt":"1"`},
		{"greater than equal to", &Operation{Operator: OperatorGreaterThanEqualTo, Operand: "1"}, `"gte":"1"`},
		{"less than", &Operation{Operator: OperatorLessThan, Operand: "1"}, `"lt":"1"`},
		{"less than equal to", &Operation{Operator: OperatorLessThanEqualTo, Operand: "1"}, `"lte":"1"`},
		{"equal to", &Operation{Operator: OperatorEqualTo, Operand: "1"}, "1"},
		// is_present takes a bare JSON boolean rather than a quoted string.
		{"is present", IsPresent(true), `"is_present":true`},
		{"is not present", IsPresent(false), `"is_present":false`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if encoded := tc.operation.Encode(); encoded != tc.expected {
				t.Fatalf("expected %s got %s", tc.expected, encoded)
			}
		})
	}
}

func TestListParams(t *testing.T) {
	t.Run("OnlyOffsetAndLimit", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})

	t.Run("OffsetLimitAndSort", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
			Sort: &Sort{
				By:   "first_name",
				Desc: true,
			},
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "sort=-first_name&offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})

	t.Run("OffsetLimitSortAndOperations", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
			Sort: &Sort{
				By:   "first_name",
				Desc: true,
			},
			Items: []*QueryItem{
				{
					Field: "created_at",
					Operations: []*Operation{
						{
							Operator: OperatorGreaterThanEqualTo,
							Operand:  "2016-05-05",
						},
						{
							Operator: OperatorLessThanEqualTo,
							Operand:  "2016-12-05",
						},
					},
				},
			},
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "created_at=%7B%22gte%22%3A%222016-05-05%22%2C%22lte%22%3A%222016-12-05%22}&sort=-first_name&offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})

	t.Run("OffsetLimitSortAndEqualTo", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
			Sort: &Sort{
				By:   "first_name",
				Desc: true,
			},
			Items: []*QueryItem{
				{
					Field: "created_at",
					Operations: []*Operation{
						{
							Operator: OperatorEqualTo,
							Operand:  "2016-05-05",
						},
					},
				},
				{
					Field: "blah",
					Operations: []*Operation{
						{
							Operator: OperatorEqualTo,
							Operand:  "N/A",
						},
					},
				},
			},
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "created_at=2016-05-05&blah=N%2FA&sort=-first_name&offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})
}
