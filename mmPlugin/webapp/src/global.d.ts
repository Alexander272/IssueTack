import type {IssuetrackWebappPlugin} from './types';

declare global {
    interface Window {
        registerPlugin: (id: string, plugin: IssuetrackWebappPlugin) => void;
    }
}

export {};