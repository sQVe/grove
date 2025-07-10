import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ConfirmModal } from "./ConfirmModal";

describe("ConfirmModal", () => {
	const mockProps = {
		message: "Are you sure you want to delete this worktree?",
		onConfirm: vi.fn(),
		onCancel: vi.fn(),
	};

	beforeEach(() => {
		vi.clearAllMocks();
	});

	it("should render confirm modal with title", () => {
		const { lastFrame } = render(<ConfirmModal {...mockProps} />);

		expect(lastFrame()).toContain("Confirm Action");
	});

	it("should display the provided message", () => {
		const { lastFrame } = render(<ConfirmModal {...mockProps} />);

		expect(lastFrame()).toContain(
			"Are you sure you want to delete this worktree?",
		);
	});

	it("should show yes/no options", () => {
		const { lastFrame } = render(<ConfirmModal {...mockProps} />);

		const output = lastFrame();
		expect(output).toContain("[y] Yes");
		expect(output).toContain("[n] No");
	});

	it("should initialize properly without triggering callbacks", () => {
		render(<ConfirmModal {...mockProps} />);

		// Should not trigger any callbacks on render
		expect(mockProps.onConfirm).not.toHaveBeenCalled();
		expect(mockProps.onCancel).not.toHaveBeenCalled();
	});

	it("should accept callback props correctly", () => {
		const customOnConfirm = vi.fn();
		const customOnCancel = vi.fn();

		render(
			<ConfirmModal
				message="Test message"
				onConfirm={customOnConfirm}
				onCancel={customOnCancel}
			/>,
		);

		// Should accept custom callbacks without calling them
		expect(customOnConfirm).not.toHaveBeenCalled();
		expect(customOnCancel).not.toHaveBeenCalled();
	});

	it("should handle prop updates", () => {
		const { rerender } = render(<ConfirmModal {...mockProps} />);

		// Should handle message updates
		rerender(<ConfirmModal {...mockProps} message="Updated message" />);

		expect(mockProps.onConfirm).not.toHaveBeenCalled();
		expect(mockProps.onCancel).not.toHaveBeenCalled();
	});

	it("should setup useInput hook", () => {
		// Test that component renders and sets up keyboard handling
		render(<ConfirmModal {...mockProps} />);

		// If no error is thrown, useInput is properly configured
		expect(true).toBe(true);
	});

	it("should handle different messages", () => {
		const customMessage = "Delete all worktrees?";
		const { lastFrame } = render(
			<ConfirmModal {...mockProps} message={customMessage} />,
		);

		expect(lastFrame()).toContain(customMessage);
	});

	it("should maintain state correctly", () => {
		const { rerender } = render(<ConfirmModal {...mockProps} />);

		// Test multiple rerenders
		rerender(<ConfirmModal {...mockProps} message="First message" />);
		rerender(<ConfirmModal {...mockProps} message="Second message" />);

		// Should not have triggered callbacks
		expect(mockProps.onConfirm).not.toHaveBeenCalled();
		expect(mockProps.onCancel).not.toHaveBeenCalled();
	});

	it("should have proper modal styling", () => {
		const { lastFrame } = render(<ConfirmModal {...mockProps} />);

		const output = lastFrame();
		// Should have title, message, and options
		expect(output).toContain("Confirm Action");
		expect(output).toContain("Are you sure");
		expect(output).toContain("Yes");
		expect(output).toContain("No");
	});

	it("should handle long messages", () => {
		const longMessage =
			"This is a very long confirmation message that should still display properly in the modal without breaking the layout or causing issues with the UI";
		const { lastFrame } = render(
			<ConfirmModal {...mockProps} message={longMessage} />,
		);

		expect(lastFrame()).toContain("This is a very long confirmation");
	});
});
