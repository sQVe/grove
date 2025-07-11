import { access, constants } from "node:fs/promises";
import { join } from "node:path";

/**
 * Naming utilities inspired by Crystal's naming patterns.
 * Provides kebab-case enforcement and uniqueness resolution.
 */

/**
 * Convert a string to kebab-case.
 * Inspired by Crystal's naming convention enforcement.
 */
export function toKebabCase(input: string): string {
	return (
		input
			.trim()
			// Replace spaces, underscores, and camelCase with hyphens
			.replace(/[\s_]+/g, "-")
			.replace(/([a-z])([A-Z])/g, "$1-$2")
			// Remove special characters except hyphens and alphanumeric
			.replace(/[^a-zA-Z0-9-]/g, "")
			// Convert to lowercase
			.toLowerCase()
			// Remove consecutive hyphens
			.replace(/-+/g, "-")
			// Remove leading/trailing hyphens
			.replace(/^-+|-+$/g, "")
	);
}

/**
 * Generate a sanitized name following Grove's naming guidelines.
 * - 2-4 words preferred
 * - Under 30 characters
 * - Kebab-case format
 * - No special characters
 */
export function sanitizeName(input: string): string {
	const kebabName = toKebabCase(input);

	// Limit to 30 characters
	if (kebabName.length > 30) {
		const words = kebabName.split("-");
		let result = "";

		// Try to include as many words as possible under the limit
		for (const word of words) {
			const testResult = result ? `${result}-${word}` : word;
			if (testResult.length <= 30) {
				result = testResult;
			} else {
				break;
			}
		}

		// If we couldn't fit any words, truncate the first word
		if (!result && words[0]) {
			result = words[0].substring(0, 30);
		}

		return result || "worktree";
	}

	return kebabName || "worktree";
}

/**
 * Check if a name exists in the given directory.
 */
export async function nameExists(
	name: string,
	directory: string,
): Promise<boolean> {
	try {
		await access(join(directory, name), constants.F_OK);
		return true;
	} catch {
		return false;
	}
}

/**
 * Generate a unique name by appending numeric suffixes.
 * Inspired by Crystal's conflict resolution approach.
 */
export async function generateUniqueName(
	baseName: string,
	directory: string,
): Promise<string> {
	const sanitizedBase = sanitizeName(baseName);

	// Check if base name is available
	if (!(await nameExists(sanitizedBase, directory))) {
		return sanitizedBase;
	}

	// Find the next available numeric suffix
	let counter = 1;
	let uniqueName: string;

	do {
		uniqueName = `${sanitizedBase}-${counter}`;
		counter++;

		// Prevent infinite loops
		if (counter > 1000) {
			throw new Error(`Unable to generate unique name for: ${baseName}`);
		}
	} while (await nameExists(uniqueName, directory));

	return uniqueName;
}

/**
 * Extract meaningful words from a string for naming.
 * Useful for generating names from prompts or descriptions.
 */
export function extractKeywords(input: string, maxWords = 4): string[] {
	// Common words to ignore
	const stopWords = new Set([
		"a",
		"an",
		"and",
		"are",
		"as",
		"at",
		"be",
		"by",
		"for",
		"from",
		"has",
		"he",
		"in",
		"is",
		"it",
		"its",
		"of",
		"on",
		"that",
		"the",
		"to",
		"was",
		"will",
		"with",
		"would",
		"could",
		"should",
		"can",
		"add",
		"create",
		"make",
		"build",
		"implement",
		"fix",
		"update",
		"change",
		"modify",
		"remove",
		"delete",
		"new",
		"old",
		"this",
		"that",
	]);

	const words = input
		.toLowerCase()
		.replace(/[^a-zA-Z0-9\s]/g, " ") // Replace non-alphanumeric with spaces
		.split(/\s+/)
		.filter((word) => word.length > 2 && !stopWords.has(word))
		.slice(0, maxWords);

	return words.length > 0 ? words : ["worktree"];
}

/**
 * Generate a worktree name from a description or prompt.
 * Combines keyword extraction with naming guidelines.
 */
export function generateWorktreeName(description: string): string {
	const keywords = extractKeywords(description, 4);
	const name = keywords.join("-");
	return sanitizeName(name);
}

/**
 * Validate that a name follows Grove's naming conventions.
 */
export function validateName(name: string): {
	valid: boolean;
	errors: string[];
} {
	const errors: string[] = [];

	if (!name) {
		errors.push("Name cannot be empty");
		return { valid: false, errors };
	}

	if (name.length > 30) {
		errors.push("Name must be 30 characters or less");
	}

	if (!/^[a-z0-9-]+$/.test(name)) {
		errors.push(
			"Name must contain only lowercase letters, numbers, and hyphens",
		);
	}

	if (name.startsWith("-") || name.endsWith("-")) {
		errors.push("Name cannot start or end with a hyphen");
	}

	if (name.includes("--")) {
		errors.push("Name cannot contain consecutive hyphens");
	}

	const wordCount = name.split("-").length;
	if (wordCount > 5) {
		errors.push("Name should have 5 words or fewer");
	}

	return { valid: errors.length === 0, errors };
}
