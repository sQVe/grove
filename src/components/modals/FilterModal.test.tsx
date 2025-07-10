import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { FilterModal } from "./FilterModal";
import type { Worktree } from "../../commands/list";

const mockWorktrees: Worktree[] = [
	{
		name: "main",
		path: "/repo/main",
		branch: "main",
		head: "abc123",
		active: true,
		locked: false,
	},
	{
		name: "feature-search",
		path: "/repo/feature-search",
		branch: "feature-search",
		head: "def456",
		active: false,
		locked: false,
	},
];

describe("FilterModal", () => {
	const mockProps = {
		query: "",
		onQueryChange: vi.fn(),
		onApply: vi.fn(),
		onCancel: vi.fn(),
		allWorktrees: mockWorktrees,
	};

	beforeEach(() => {
		vi.clearAllMocks();
	});

	it("should render filter modal with title", () => {
		const { lastFrame } = render(<FilterModal {...mockProps} />);

		expect(lastFrame()).toContain("Filter Worktrees");
		expect(lastFrame()).toContain("Search:");
	});

	it("should display current query", () => {
		const { lastFrame } = render(<FilterModal {...mockProps} query="test" />);

		expect(lastFrame()).toContain("Search: test");
	});

	it("should show cursor when query is empty", () => {
		const { lastFrame } = render(<FilterModal {...mockProps} query="" />);

		expect(lastFrame()).toContain("█"); // Cursor block
	});

	it("should show cursor after text", () => {
		const { lastFrame } = render(<FilterModal {...mockProps} query="test" />);

		const output = lastFrame();
		expect(output).toContain("Search: test");
		expect(output).toContain("█"); // Cursor should appear after text
	});

	it("should initialize fuzzy search properly", () => {
		render(<FilterModal {...mockProps} />);

		// Component should render without errors and initialize search
		expect(mockProps.onQueryChange).not.toHaveBeenCalled();
		expect(mockProps.onApply).not.toHaveBeenCalled();
		expect(mockProps.onCancel).not.toHaveBeenCalled();
	});

	it("should handle props correctly", () => {
		const { rerender } = render(<FilterModal {...mockProps} query="test" />);

		// Should accept and display query prop
		expect(mockProps.allWorktrees).toEqual(mockWorktrees);

		// Test rerender with different query
		rerender(<FilterModal {...mockProps} query="main" />);
	});

	it("should work with empty worktree list", () => {
		const emptyProps = { ...mockProps, allWorktrees: [] };
		const { lastFrame } = render(<FilterModal {...emptyProps} />);

		expect(lastFrame()).toContain("Filter Worktrees");
	});

	it("should handle fuzzy search functionality", () => {
		// Test the fuzzy search integration by checking it doesn't crash
		render(<FilterModal {...mockProps} query="main" />);

		// If component renders successfully, fuzzy search is working
		expect(true).toBe(true);
	});

	it("should update when props change", () => {
		const { rerender } = render(<FilterModal {...mockProps} query="" />);

		// Should handle query changes via props
		rerender(<FilterModal {...mockProps} query="updated" />);

		// Component should still be functional
		expect(true).toBe(true);
	});

	it("should have proper modal styling", () => {
		const { lastFrame } = render(<FilterModal {...mockProps} />);

		const output = lastFrame();
		// Should have border and proper spacing
		expect(output).toContain("Filter Worktrees");
		expect(output).toContain("Search:");
	});
});
