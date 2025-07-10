import React from "react";
import { render } from "ink";
import { App } from "./App";

export async function launchTUI(): Promise<void> {
	const { waitUntilExit } = render(<App />);
	await waitUntilExit();
}
