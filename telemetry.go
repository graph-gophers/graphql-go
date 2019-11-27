package graphql

import (
	"fmt"

	"github.com/JoinCAD/graphql-go/errors"
	"github.com/JoinCAD/graphql-go/internal/common"
	"github.com/JoinCAD/graphql-go/internal/query"
	"github.com/JoinCAD/graphql-go/internal/validation"
)

// LoggedOperation represents a summary of an operation suitable for concise
// telemetry, for example in a web server context.
type LoggedOperation struct {
	Name      string `json:",omitempty"`
	Type      query.OperationType
	Variables map[string]string `json:",omitempty"`
	Fields    []LoggedField     `json:",omitempty"`
}

// LoggedField represents a summary of a field.
type LoggedField struct {
	Name      string
	Arguments map[string]string `json:",omitempty"`
}

func logField(field query.Field) LoggedField {
	args := []common.Argument(field.Arguments)
	var loggedArgs map[string]string
	if len(args) > 0 {
		loggedArgs = make(map[string]string)
		for _, arg := range args {
			loggedArgs[arg.Name.Name] = arg.Value.String()
		}
	}
	return LoggedField{
		Name:      field.Name.Name,
		Arguments: loggedArgs,
	}
}

func logOperations(doc *query.Document) []LoggedOperation {
	ops := []*query.Operation(doc.Operations)
	lops := make([]LoggedOperation, len(ops))
	for i, op := range ops {
		inputs := []*common.InputValue(op.Vars)
		var args map[string]string
		if len(inputs) > 0 {
			args = make(map[string]string)
			for _, input := range inputs {
				if input != nil && input.Default != nil {
					args[input.Name.Name] = input.Default.String()
				}
			}
		}

		fields := make([]LoggedField, 0, len(op.Selections))
		for _, sel := range op.Selections {
			fmt.Printf("%+v\n", sel)
			if field, ok := sel.(query.Field); ok {
				fields = append(fields, logField(field))
			} else if field, ok := sel.(*query.Field); ok {
				fields = append(fields, logField(*field))
			}
		}

		lops[i] = LoggedOperation{
			Name:      op.Name.Name,
			Type:      op.Type,
			Variables: args,
			Fields:    fields,
		}
	}
	return lops
}

// ValidateAndLog validates the query and simultaneously produces a loggable
// summary of the operations it contains.
func (s *Schema) ValidateAndLog(queryString string, variables map[string]interface{}) ([]*errors.QueryError, []LoggedOperation) {
	doc, qErr := query.Parse(queryString)
	if qErr != nil {
		return []*errors.QueryError{qErr}, nil
	}

	validationFinish := s.validationTracer.TraceValidation()
	errs := validation.Validate(s.schema, doc, variables, s.maxDepth)
	validationFinish(errs)
	if len(errs) != 0 {
		return errs, []LoggedOperation{}
	}
	return errs, logOperations(doc)
}
