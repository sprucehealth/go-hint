package hint

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Params is an interface that any object that implements the validate method
// conforms to
type Params interface {
	Validate() error
}

// ListParams represents the parameters for querying through a list of paginated resources.
type ListParams struct {
	Items  []*QueryItem
	Sort   *Sort
	Offset uint64
	Limit  uint64
}

// Sort represents the fields to configure a sort by while querying for a list of resources.
type Sort struct {
	By   string
	Desc bool
}

// QueryItem represents an operation on a particular field of the resource while querying
// for a list of resources
type QueryItem struct {
	Field      string
	Operations []*Operation
}

// Operation represents a logical filter to apply when querying for a list of resources.
type Operation struct {
	Operator Operator
	Operand  string
}

// equalToQueryItem builds the QueryItem for an exact match on a field, the
// shape typed list params use for their simple string filters.
func equalToQueryItem(field, value string) *QueryItem {
	return &QueryItem{
		Field:      field,
		Operations: []*Operation{{Operator: OperatorEqualTo, Operand: value}},
	}
}

type Operator int

const (
	OperatorGreaterThan Operator = 1 << iota
	OperatorGreaterThanEqualTo
	OperatorLessThan
	OperatorLessThanEqualTo
	OperatorEqualTo
	// OperatorIsPresent matches on whether the field is set at all rather than on
	// its value. It is only accepted on select filters, such as the
	// deactivated_at filter on an installation's API keys. Build it with
	// IsPresent, which formats the operand correctly.
	OperatorIsPresent
)

// IsPresent returns the operation that narrows a filter to records where the
// field is set (present true) or unset (present false).
func IsPresent(present bool) *Operation {
	return &Operation{Operator: OperatorIsPresent, Operand: strconv.FormatBool(present)}
}

func (l *ListParams) Encode() (string, error) {
	var buffer bytes.Buffer

	if len(l.Items) > 0 {
		for _, item := range l.Items {
			if buffer.Len() > 0 {
				buffer.WriteByte('&')
			}

			encoded, err := item.Encode()
			if err != nil {
				return "", err
			}
			buffer.WriteString(encoded)
		}
	}

	if l.Sort != nil {
		if buffer.Len() > 0 {
			buffer.WriteByte('&')
		}
		buffer.WriteString(l.Sort.Encode())
	}

	if l.Offset > 0 {
		if buffer.Len() > 0 {
			buffer.WriteByte('&')
		}
		buffer.WriteString("offset=")
		buffer.WriteString(strconv.FormatUint(uint64(l.Offset), 10))
	}

	if l.Limit > 0 {
		if buffer.Len() > 0 {
			buffer.WriteByte('&')
		}
		buffer.WriteString("limit=")
		buffer.WriteString(strconv.FormatUint(uint64(l.Limit), 10))
	}

	return buffer.String(), nil
}

func (q *Sort) Encode() string {
	var buffer bytes.Buffer
	buffer.WriteString("sort")
	buffer.WriteString("=")
	if q.Desc {
		buffer.WriteString("-")
	}
	buffer.WriteString(url.QueryEscape(q.By))
	return buffer.String()
}

func (q *QueryItem) Encode() (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString(url.QueryEscape(q.Field))
	buffer.WriteString("=")

	if len(q.Operations) > 1 && containsEqualToOperator(q.Operations) {
		return "", errors.New("cannot have multiple operations when there is an equal to operator")
	}

	if len(q.Operations) == 1 && q.Operations[0].Operator == OperatorEqualTo {
		buffer.WriteString(url.QueryEscape(q.Operations[0].Encode()))
	} else {
		encodedOperations := make([]string, len(q.Operations))
		for i, operation := range q.Operations {
			encodedOperations[i] = operation.Encode()
		}
		buffer.WriteString(url.QueryEscape("{"+strings.Join(encodedOperations, ",")) + "}")
	}

	return buffer.String(), nil
}

func (o *Operation) Encode() string {
	switch o.Operator {
	case OperatorGreaterThan:
		return fmt.Sprintf(`"gt":"%s"`, o.Operand)
	case OperatorLessThan:
		return fmt.Sprintf(`"lt":"%s"`, o.Operand)
	case OperatorGreaterThanEqualTo:
		return fmt.Sprintf(`"gte":"%s"`, o.Operand)
	case OperatorLessThanEqualTo:
		return fmt.Sprintf(`"lte":"%s"`, o.Operand)
	case OperatorEqualTo:
		return o.Operand
	case OperatorIsPresent:
		// The operand is a JSON boolean here rather than a quoted string.
		return fmt.Sprintf(`"is_present":%s`, o.Operand)
	}

	return ""
}

func containsEqualToOperator(operations []*Operation) bool {
	for _, operation := range operations {
		if operation.Operator == OperatorEqualTo {
			return true
		}
	}
	return false
}
