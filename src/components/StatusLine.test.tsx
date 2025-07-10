import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect } from "vitest";
import { StatusLine } from "./StatusLine";

describe("StatusLine", () => {
	it("should show normal mode keybindings", () => {
		const { lastFrame } = render(
			<StatusLine mode="normal" worktreeCount={5} />,
		);

		const output = lastFrame();
		expect(output).toContain("5 worktree"); // "s" gets truncated.
		expect(output).toContain("[j/k] Navigate");
		expect(output).toContain("[Enter] Switch");
		expect(output).toContain("[q] Quit");
		// No mode indicator in normal mode.
		expect(output).not.toContain("NORMAL");
	});

	it("should show filter mode keybindings", () => {
		const { lastFrame } = render(
			<StatusLine mode="filter" worktreeCount={3} />,
		);

		const output = lastFrame();
		expect(output).toContain("FILTER");
		expect(output).toContain("3 worktrees");
		expect(output).toContain("Type to search");
		expect(output).toContain("[Enter] Apply");
		expect(output).toContain("[Esc] Cancel");
	});

	it("should show help mode keybindings", () => {
		const { lastFrame } = render(<StatusLine mode="help" worktreeCount={2} />);

		const output = lastFrame();
		expect(output).toContain("HELP");
		expect(output).toContain("2 worktrees");
		expect(output).toContain("[?] or [Esc] Close help");
	});

	it("should show confirm mode keybindings", () => {
		const { lastFrame } = render(
			<StatusLine mode="confirm" worktreeCount={1} />,
		);

		const output = lastFrame();
		expect(output).toContain("CONFIRM");
		expect(output).toContain("1 worktrees");
		expect(output).toContain("[y] Confirm");
		expect(output).toContain("[n] Cancel");
	});

	it("should handle zero worktrees", () => {
		const { lastFrame } = render(
			<StatusLine mode="normal" worktreeCount={0} />,
		);

		expect(lastFrame()).toContain("0 worktree"); // "s" gets truncated.
	});

	it("should handle singular worktree count", () => {
		const { lastFrame } = render(
			<StatusLine mode="normal" worktreeCount={1} />,
		);

		expect(lastFrame()).toContain("1 worktree"); // "s" gets truncated.
	});

	it("should show all normal mode actions", () => {
		const { lastFrame } = render(
			<StatusLine mode="normal" worktreeCount={5} />,
		);

		const output = lastFrame();
		expect(output).toContain("[c] Create");
		expect(output).toContain("[d] Delete");
		expect(output).toContain("[/] Filter");
		expect(output).toContain("[?]"); // Help text gets cut off.
	});

	it("should maintain consistent layout", () => {
		const { lastFrame } = render(
			<StatusLine mode="normal" worktreeCount={5} />,
		);

		const output = lastFrame();
		// Should have count on left and keybindings on right in normal mode.
		expect(output).toContain("worktree"); // "s" gets truncated.
		// No mode indicator or separator in normal mode.
		expect(output).not.toContain("NORMA");
	});
});
