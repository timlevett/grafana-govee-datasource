// Ensure React loads its development build so that act() and other dev-only
// APIs are available. Jest should set NODE_ENV=test automatically, but some
// environments do not propagate it correctly before module resolution.
process.env.NODE_ENV = 'test';
