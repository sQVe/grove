// Test setup file
import { vi } from "vitest";

// Mock console methods to avoid noise in tests
vi.stubGlobal("console", {
	...console,
	log: vi.fn(),
	error: vi.fn(),
	warn: vi.fn(),
});

// Set up global DOM environment for React testing
Object.defineProperty(globalThis, "TextEncoder", {
	value: TextEncoder,
});

Object.defineProperty(globalThis, "TextDecoder", {
	value: TextDecoder,
});
