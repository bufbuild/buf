package protodesc

import "fmt"

type method struct {
	namedDescriptor

	service              Service
	inputTypeName        string
	outputTypeName       string
	clientStreaming      bool
	serverStreaming      bool
	inputTypePath        []int32
	outputTypePath       []int32
	idempotencyLevel     MethodOptionsIdempotencyLevel
	idempotencyLevelPath []int32
}

func newMethod(
	namedDescriptor namedDescriptor,
	service Service,
	inputTypeName string,
	outputTypeName string,
	clientStreaming bool,
	serverStreaming bool,
	inputTypePath []int32,
	outputTypePath []int32,
	idempotencyLevel MethodOptionsIdempotencyLevel,
	idempotencyLevelPath []int32,
) (*method, error) {
	if inputTypeName == "" {
		return nil, fmt.Errorf("no inputTypeName on %q", namedDescriptor.name)
	}
	if outputTypeName == "" {
		return nil, fmt.Errorf("no outputTypeName on %q", namedDescriptor.name)
	}
	return &method{
		namedDescriptor:      namedDescriptor,
		service:              service,
		inputTypeName:        inputTypeName,
		outputTypeName:       outputTypeName,
		clientStreaming:      clientStreaming,
		serverStreaming:      serverStreaming,
		inputTypePath:        inputTypePath,
		outputTypePath:       outputTypePath,
		idempotencyLevel:     idempotencyLevel,
		idempotencyLevelPath: idempotencyLevelPath,
	}, nil
}

func (m *method) Service() Service {
	return m.service
}

func (m *method) InputTypeName() string {
	return m.inputTypeName
}

func (m *method) OutputTypeName() string {
	return m.outputTypeName
}

func (m *method) ClientStreaming() bool {
	return m.clientStreaming
}

func (m *method) ServerStreaming() bool {
	return m.serverStreaming
}

func (m *method) InputTypeLocation() Location {
	return m.getLocation(m.inputTypePath)
}

func (m *method) OutputTypeLocation() Location {
	return m.getLocation(m.outputTypePath)
}

func (m *method) IdempotencyLevel() MethodOptionsIdempotencyLevel {
	return m.idempotencyLevel
}

func (m *method) IdempotencyLevelLocation() Location {
	return m.getLocation(m.idempotencyLevelPath)
}
