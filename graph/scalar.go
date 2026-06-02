package graph

import (
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalTime(t time.Time) graphql.Marshaler {
	return graphql.MarshalTime(t)
}

func UnmarshalTime(v interface{}) (time.Time, error) {
	return graphql.UnmarshalTime(v)
}

func MarshalID(s string) graphql.Marshaler {
	return graphql.MarshalString(s)
}

func UnmarshalID(v interface{}) (string, error) {
	return graphql.UnmarshalString(v)
}

func MarshalAny(v interface{}) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(`null`))
	})
}
