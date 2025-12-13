// Package prompty provides a dynamic LLM prompt templating system with plugin architecture.
//
// Prompty uses a simple tag syntax with {~ and ~} delimiters for dynamic content:
//
//	Hello, {~prompty.var name="user" /~}!
//
// # Basic Usage
//
// Create an engine and execute templates:
//
//	engine := prompty.MustNew()
//	result, err := engine.Execute(ctx, "Hello, {~prompty.var name=\"user\" /~}!", map[string]any{
//	    "user": "Alice",
//	})
//	// result: "Hello, Alice!"
//
// # Template Syntax
//
// Prompty supports two types of tags:
//
// Self-closing tags: {~tagname attr="value" /~}
//
//	{~prompty.var name="user" default="Guest" /~}
//
// Block tags: {~tagname~}content{~/tagname~}
//
//	{~section~}This is a section{~/section~}
//
// # Built-in Tags
//
// prompty.var - Variable interpolation with optional default:
//
//	{~prompty.var name="user.name" default="Anonymous" /~}
//
// prompty.raw - Preserve literal content (no parsing):
//
//	{~prompty.raw~}{{ jinja_syntax }}{~/prompty.raw~}
//
// # Custom Resolvers
//
// Extend prompty with custom tag handlers by implementing the Resolver interface:
//
//	type MyResolver struct{}
//
//	func (r *MyResolver) TagName() string { return "myapp.greet" }
//
//	func (r *MyResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
//	    name, _ := attrs.Get("name")
//	    return "Hello, " + name + "!", nil
//	}
//
//	func (r *MyResolver) Validate(attrs prompty.Attributes) error {
//	    if !attrs.Has("name") { return errors.New("missing name") }
//	    return nil
//	}
//
//	// Register and use:
//	engine.MustRegister(&MyResolver{})
//	result, _ := engine.Execute(ctx, "{~myapp.greet name=\"World\" /~}", nil)
//
// # Error Handling
//
// Prompty uses the throw error strategy by default, returning errors immediately.
// Errors include position information for debugging:
//
//	result, err := engine.Execute(ctx, template, data)
//	if err != nil {
//	    // err contains line/column information
//	}
//
// # Configuration
//
// Customize the engine with functional options:
//
//	engine, _ := prompty.New(
//	    prompty.WithDelimiters("<%", "%>"),
//	    prompty.WithMaxDepth(50),
//	    prompty.WithLogger(logger),
//	)
package prompty
