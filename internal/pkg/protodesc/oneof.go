package protodesc

type oneof struct {
	namedDescriptor

	message Message
}

func newOneof(
	namedDescriptor namedDescriptor,
	message Message,
) *oneof {
	return &oneof{
		namedDescriptor: namedDescriptor,
		message:         message,
	}
}

func (o *oneof) Message() Message {
	return o.message
}
