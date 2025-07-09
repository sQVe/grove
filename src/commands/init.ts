export interface InitOptions {
	path?: string;
	bare?: boolean;
	defaultBranch?: string;
}

export async function initRepository({
	path = process.cwd(),
	bare = true,
	defaultBranch = "main",
}: InitOptions = {}) {
	const {
		initBareRepository,
		isGitAvailable,
		isGitRepository,
		validatePath,
		GitOperationError,
	} = await import("../lib/git.js");
	const { mkdir } = await import("node:fs/promises");
	const { resolve, join } = await import("node:path");

	try {
		// Check if git is available.
		if (!(await isGitAvailable())) {
			throw new Error("Git is not installed or not available in PATH");
		}

		const resolvedPath = resolve(path);

		// Check if path already exists and is a git repository.
		try {
			await validatePath(resolvedPath);
			if (await isGitRepository(resolvedPath)) {
				throw new Error(
					`Directory ${resolvedPath} is already a git repository`,
				);
			}
		} catch (error) {
			// Path doesn't exist, create it.
			if (error instanceof GitOperationError && error.code === "PATH_ERROR") {
				await mkdir(resolvedPath, { recursive: true });
			} else {
				throw error;
			}
		}

		if (bare) {
			// Initialize bare repository.
			await initBareRepository(resolvedPath, defaultBranch);

			// Create grove configuration directory.
			const groveConfigDir = join(resolvedPath, ".grove");
			await mkdir(groveConfigDir, { recursive: true });

			console.log(`✓ Initialized bare repository at: ${resolvedPath}`);
			console.log(`✓ Default branch: ${defaultBranch}`);
			console.log("✓ Grove configuration directory created");
		} else {
			throw new Error("Non-bare repository initialization not yet supported");
		}
	} catch (error) {
		if (error instanceof GitOperationError) {
			throw new Error(`Failed to initialize repository: ${error.message}`);
		}
		throw error;
	}
}
