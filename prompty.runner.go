package prompty

// TemplateRunner is the common interface for resolver management shared by Engine and StorageEngine.
// Both Engine and StorageEngine implement this interface, allowing generic code to work with either.
//
// Note: Execution signatures differ between Engine (source string) and StorageEngine (template name),
// so this interface covers resolver registration and introspection only.
type TemplateRunner interface {
	// RegisterResolver adds a custom resolver to the engine.
	// Returns an error if a resolver for the same tag name is already registered.
	RegisterResolver(r Resolver) error

	// HasResolver checks if a resolver is registered for the given tag name.
	HasResolver(tagName string) bool

	// ListResolvers returns all registered resolver tag names in sorted order.
	ListResolvers() []string

	// ResolverCount returns the number of registered resolvers.
	ResolverCount() int
}

// Compile-time checks that both Engine and StorageEngine satisfy TemplateRunner.
var (
	_ TemplateRunner = (*Engine)(nil)
	_ TemplateRunner = (*StorageEngine)(nil)
)
