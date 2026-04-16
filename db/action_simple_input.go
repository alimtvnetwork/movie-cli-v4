// action_simple_input.go — ActionSimpleInput option struct for InsertActionSimple.
package db

// ActionSimpleInput groups parameters for the InsertActionSimple convenience wrapper.
type ActionSimpleInput struct {
	FileAction FileActionType
	MediaID    int64
	Snapshot   string
	Detail     string
	BatchID    string
}
