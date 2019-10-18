package protodesc

type service struct {
	namedDescriptor

	methods []Method
}

func newService(
	namedDescriptor namedDescriptor,
) *service {
	return &service{
		namedDescriptor: namedDescriptor,
	}
}

func (m *service) Methods() []Method {
	return m.methods
}

func (m *service) addMethod(method Method) {
	m.methods = append(m.methods, method)
}
