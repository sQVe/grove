import { cosmiconfig } from "cosmiconfig";
import { z } from "zod";

export const configSchema = z.object({
	defaultBranch: z.string().default("main"),
	autoFetch: z.boolean().default(true),
	confirmDestructive: z.boolean().default(true),
	tui: z
		.object({
			theme: z.string().default("default"),
			vimBindings: z.boolean().default(true),
			previewEnabled: z.boolean().default(true),
		})
		.default({}),
	env: z
		.object({
			files: z
				.array(z.string())
				.default([".env", ".env.local", "package.json"]),
			ignorePatterns: z
				.array(z.string())
				.default(["node_modules", ".git", "dist"]),
		})
		.default({}),
});

export type Config = z.infer<typeof configSchema>;

export async function loadConfig(): Promise<Config> {
	const explorer = cosmiconfig("grove");
	const result = await explorer.search();

	const rawConfig = result?.config || {};
	return configSchema.parse(rawConfig);
}

export async function saveConfig(config: Config): Promise<void> {
	throw new Error("saveConfig not implemented yet");
}
