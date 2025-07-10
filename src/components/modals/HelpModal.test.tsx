import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect } from "vitest";
import { HelpModal } from "./HelpModal";

describe("HelpModal", () => {
	it("should render help modal with title", () => {
		const { lastFrame } = render(<HelpModal />);

		expect(lastFrame()).toContain("Grove - Keyboard Shortcuts");
	});

	it("should show navigation shortcuts", () => {
		const { lastFrame } = render(<HelpModal />);

		const output = lastFrame();
		expect(output).toContain("Navigation:");
		expect(output).toContain("j / ↓");
		expect(output).toContain("k / ↑");
		expect(output).toContain("Enter");
		expect(output).toContain("Move down");
		expect(output).toContain("Move up");
		expect(output).toContain("Switch to worktree");
	});

	it("should show action shortcuts", () => {
		const { lastFrame } = render(<HelpModal />);

		const output = lastFrame();
		expect(output).toContain("Actions:");
		expect(output).toContain("c");
		expect(output).toContain("d");
		expect(output).toContain("r");
		expect(output).toContain("Create new worktree");
		expect(output).toContain("Delete worktree");
		expect(output).toContain("Rename worktree");
	});

	it("should show search and help shortcuts", () => {
		const { lastFrame } = render(<HelpModal />);

		const output = lastFrame();
		expect(output).toContain("Search & Help:");
		expect(output).toContain("/");
		expect(output).toContain("?");
		expect(output).toContain("Filter worktrees");
		expect(output).toContain("Toggle this help");
	});

	it("should show exit shortcuts", () => {
		const { lastFrame } = render(<HelpModal />);

		const output = lastFrame();
		expect(output).toContain("Exit:");
		expect(output).toContain("q");
		expect(output).toContain("Esc");
		expect(output).toContain("Quit Grove");
		expect(output).toContain("Cancel current action");
	});

	it("should show instructions to close help", () => {
		const { lastFrame } = render(<HelpModal />);

		const output = lastFrame();
		expect(output).toContain("Press");
		expect(output).toContain("to close this help");
	});

	it("should have proper categories structure", () => {
		const { lastFrame } = render(<HelpModal />);

		const output = lastFrame();
		// Should have all main categories
		expect(output).toContain("Navigation:");
		expect(output).toContain("Actions:");
		expect(output).toContain("Search & Help:");
		expect(output).toContain("Exit:");
	});

	it("should display all keybindings mentioned in StatusLine", () => {
		const { lastFrame } = render(<HelpModal />);

		const output = lastFrame();
		// Cross-reference with StatusLine normal mode keybindings
		expect(output).toContain("j"); // Navigate
		expect(output).toContain("k"); // Navigate
		expect(output).toContain("Enter"); // Switch
		expect(output).toContain("c"); // Create
		expect(output).toContain("d"); // Delete
		expect(output).toContain("/"); // Filter
		expect(output).toContain("?"); // Help
		expect(output).toContain("q"); // Quit
	});
});
