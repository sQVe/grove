import Fuse from "fuse.js";
import type { Worktree } from "../commands/list.js";

export interface FuzzySearchOptions {
	threshold?: number;
	includeScore?: boolean;
	minMatchCharLength?: number;
}

export class WorktreeFuzzySearch {
	private fuse: Fuse<Worktree>;
	private allWorktrees: Worktree[];

	constructor(worktrees: Worktree[], options: FuzzySearchOptions = {}) {
		this.allWorktrees = worktrees;
		this.fuse = new Fuse(worktrees, {
			keys: [
				{ name: "name", weight: 0.7 },
				{ name: "branch", weight: 0.5 },
				{ name: "path", weight: 0.3 },
			],
			threshold: options.threshold ?? 0.4,
			includeScore: options.includeScore ?? true,
			minMatchCharLength: options.minMatchCharLength ?? 1,
		});
	}

	search(query: string): Worktree[] {
		if (!query) {
			return this.allWorktrees;
		}

		return this.fuse.search(query).map((result) => result.item);
	}

	update(worktrees: Worktree[]): void {
		this.allWorktrees = worktrees;
		this.fuse.setCollection(worktrees);
	}
}
