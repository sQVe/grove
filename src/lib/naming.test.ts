import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mkdtemp, rm, mkdir } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
	toKebabCase,
	sanitizeName,
	nameExists,
	generateUniqueName,
	extractKeywords,
	generateWorktreeName,
	validateName,
} from "./naming.js";

describe("Naming utilities", () => {
	describe("toKebabCase", () => {
		it("should convert camelCase to kebab-case", () => {
			expect(toKebabCase("camelCase")).toBe("camel-case");
			expect(toKebabCase("myVariableName")).toBe("my-variable-name");
		});

		it("should convert spaces to hyphens", () => {
			expect(toKebabCase("hello world")).toBe("hello-world");
			expect(toKebabCase("multiple   spaces")).toBe("multiple-spaces");
		});

		it("should convert underscores to hyphens", () => {
			expect(toKebabCase("snake_case")).toBe("snake-case");
			expect(toKebabCase("multiple__underscores")).toBe("multiple-underscores");
		});

		it("should remove special characters", () => {
			expect(toKebabCase("hello@world!")).toBe("helloworld");
			expect(toKebabCase("test#123$")).toBe("test123");
		});

		it("should handle mixed formats", () => {
			expect(toKebabCase("My_Variable Name")).toBe("my-variable-name");
			expect(toKebabCase("camelCase_with spaces")).toBe(
				"camel-case-with-spaces",
			);
		});

		it("should remove leading and trailing hyphens", () => {
			expect(toKebabCase("-leading")).toBe("leading");
			expect(toKebabCase("trailing-")).toBe("trailing");
			expect(toKebabCase("-both-")).toBe("both");
		});

		it("should handle empty strings", () => {
			expect(toKebabCase("")).toBe("");
			expect(toKebabCase("   ")).toBe("");
		});
	});

	describe("sanitizeName", () => {
		it("should apply kebab-case formatting", () => {
			expect(sanitizeName("Hello World")).toBe("hello-world");
		});

		it("should limit to 30 characters", () => {
			const longName =
				"this-is-a-very-long-name-that-exceeds-thirty-characters";
			const result = sanitizeName(longName);
			expect(result.length).toBeLessThanOrEqual(30);
		});

		it("should preserve as many words as possible under 30 chars", () => {
			const result = sanitizeName("one two three four five six seven");
			expect(result).toBe("one-two-three-four-five-six");
			expect(result.length).toBeLessThanOrEqual(30);
		});

		it("should handle single long word", () => {
			const result = sanitizeName("supercalifragilisticexpialidocious");
			expect(result.length).toBeLessThanOrEqual(30);
			expect(result).toBe("supercalifragilisticexpialidoc");
		});

		it("should return default for empty input", () => {
			expect(sanitizeName("")).toBe("worktree");
			expect(sanitizeName("   ")).toBe("worktree");
			expect(sanitizeName("@#$%")).toBe("worktree");
		});
	});

	describe("nameExists", () => {
		let tempDir: string;

		beforeEach(async () => {
			tempDir = await mkdtemp(join(tmpdir(), "grove-test-"));
		});

		afterEach(async () => {
			await rm(tempDir, { recursive: true });
		});

		it("should return true for existing directory", async () => {
			await mkdir(join(tempDir, "existing"));
			const exists = await nameExists("existing", tempDir);
			expect(exists).toBe(true);
		});

		it("should return false for non-existing directory", async () => {
			const exists = await nameExists("non-existing", tempDir);
			expect(exists).toBe(false);
		});
	});

	describe("generateUniqueName", () => {
		let tempDir: string;

		beforeEach(async () => {
			tempDir = await mkdtemp(join(tmpdir(), "grove-test-"));
		});

		afterEach(async () => {
			await rm(tempDir, { recursive: true });
		});

		it("should return base name if available", async () => {
			const result = await generateUniqueName("test", tempDir);
			expect(result).toBe("test");
		});

		it("should append numeric suffix for conflicts", async () => {
			await mkdir(join(tempDir, "test"));
			await mkdir(join(tempDir, "test-1"));

			const result = await generateUniqueName("test", tempDir);
			expect(result).toBe("test-2");
		});

		it("should handle multiple conflicts", async () => {
			await mkdir(join(tempDir, "feature"));
			await mkdir(join(tempDir, "feature-1"));
			await mkdir(join(tempDir, "feature-2"));
			await mkdir(join(tempDir, "feature-3"));

			const result = await generateUniqueName("feature", tempDir);
			expect(result).toBe("feature-4");
		});

		it("should sanitize the base name", async () => {
			const result = await generateUniqueName("Test Feature!", tempDir);
			expect(result).toBe("test-feature");
		});
	});

	describe("extractKeywords", () => {
		it("should extract meaningful words", () => {
			const result = extractKeywords("Add user authentication feature");
			expect(result).toEqual(["user", "authentication", "feature"]);
		});

		it("should filter out stop words", () => {
			const result = extractKeywords(
				"The quick brown fox jumps over the lazy dog",
			);
			expect(result).toEqual(["quick", "brown", "fox", "jumps"]);
		});

		it("should limit to maxWords", () => {
			const result = extractKeywords("one two three four five six", 3);
			expect(result).toHaveLength(3);
			expect(result).toEqual(["one", "two", "three"]);
		});

		it("should handle short words and special characters", () => {
			const result = extractKeywords("Fix bug in API & UI components!");
			expect(result).toEqual(["bug", "api", "components"]);
		});

		it("should return default for empty input", () => {
			expect(extractKeywords("")).toEqual(["worktree"]);
			expect(extractKeywords("a an the")).toEqual(["worktree"]);
		});
	});

	describe("generateWorktreeName", () => {
		it("should generate name from description", () => {
			const result = generateWorktreeName("Add user authentication system");
			expect(result).toBe("user-authentication-system");
		});

		it("should handle long descriptions", () => {
			const result = generateWorktreeName(
				"Implement a comprehensive user authentication and authorization system with JWT tokens",
			);
			expect(result.length).toBeLessThanOrEqual(30);
		});

		it("should handle special characters", () => {
			const result = generateWorktreeName("Fix bug #123: API error handling");
			expect(result).toBe("bug-123-api-error");
		});

		it("should handle empty description", () => {
			const result = generateWorktreeName("");
			expect(result).toBe("worktree");
		});
	});

	describe("validateName", () => {
		it("should accept valid kebab-case names", () => {
			const result = validateName("feature-auth");
			expect(result.valid).toBe(true);
			expect(result.errors).toHaveLength(0);
		});

		it("should accept names with numbers", () => {
			const result = validateName("feature-v2");
			expect(result.valid).toBe(true);
		});

		it("should reject empty names", () => {
			const result = validateName("");
			expect(result.valid).toBe(false);
			expect(result.errors).toContain("Name cannot be empty");
		});

		it("should reject names that are too long", () => {
			const result = validateName(
				"this-is-a-very-long-name-that-exceeds-the-thirty-character-limit",
			);
			expect(result.valid).toBe(false);
			expect(result.errors).toContain("Name must be 30 characters or less");
		});

		it("should reject names with uppercase letters", () => {
			const result = validateName("Feature-Auth");
			expect(result.valid).toBe(false);
			expect(result.errors).toContain(
				"Name must contain only lowercase letters, numbers, and hyphens",
			);
		});

		it("should reject names with special characters", () => {
			const result = validateName("feature@auth");
			expect(result.valid).toBe(false);
			expect(result.errors).toContain(
				"Name must contain only lowercase letters, numbers, and hyphens",
			);
		});

		it("should reject names starting or ending with hyphens", () => {
			expect(validateName("-feature").valid).toBe(false);
			expect(validateName("feature-").valid).toBe(false);
		});

		it("should reject names with consecutive hyphens", () => {
			const result = validateName("feature--auth");
			expect(result.valid).toBe(false);
			expect(result.errors).toContain(
				"Name cannot contain consecutive hyphens",
			);
		});

		it("should warn about too many words", () => {
			const result = validateName("one-two-three-four-five-six");
			expect(result.valid).toBe(false);
			expect(result.errors).toContain("Name should have 5 words or fewer");
		});
	});
});
