export type Screenshot = { dataUrl: string; width: number; height: number };
export type CapturedScreenshot = Screenshot & { id: number };
export type CaptureEvent = { kind: 'started' | 'done' | 'cancelled' | 'error'; image?: Screenshot; error?: string };
export type StreamEvent = { requestId: string; kind: 'text' | 'done' | 'cancelled' | 'error'; text?: string; error?: string; usage?: { inputTokens: number; outputTokens: number } };
export type PrototypeStatus = {
  alwaysOnTop: boolean;
  captureProtectionRequested: boolean;
  shortcut: string;
  shortcutRegistered: boolean;
  shortcutError?: string;
 captureShortcut: string;
 captureShortcutRegistered: boolean;
 captureShortcutError?: string;
};
export type Config = { captureChecksEnabled: boolean; alwaysOnTop: boolean; version: number; provider: string; model: string };
export type Problem = { screenshots: string[]; statement: string; code: string; examples: string; language: string; action: string; testInput: string; expected: string; actual: string };
type Desktop = {
  GetPreferences(): Promise<{ config: Config; keyAvailable: boolean; imageSupported: boolean; message?: string }>;
  SetCaptureChecks(enabled: boolean): Promise<Config>;
  SavePreferences(config: Config): Promise<void>;
  DeleteKey(provider: string): Promise<void>;
  SaveKey(provider: string, key: string): Promise<void>;
  StartResponse(problem: Problem): Promise<string>;
  AskFollowup(question: string): Promise<string>;
  CaptureScreenshot(mode: 'screen' | 'region'): Promise<void>;
  GetStatus(): Promise<PrototypeStatus>;
  StartDemo(marker: string): Promise<string>;
  CancelDemo(): Promise<void>;
  ResetDemo(): Promise<void>;
  Cancel(): Promise<void>;
  Reset(): Promise<void>;
  SetAlwaysOnTop(enabled: boolean): Promise<void>;
  HideTemporarily(): Promise<void>;
};
declare global {
  interface Window {
    go?: { main: { Desktop: Desktop } };
    runtime?: { EventsOn(name: string, callback: (event: any) => void): () => void };
  }
}
export function desktop(): Desktop {
  if (!window.go) throw new Error('Desktop controls require the packaged app. Browser preview shows the layout only.');
  return window.go.main.Desktop;
}
export function onEvent<T>(name: string, callback: (event: T) => void): () => void {
  return window.runtime?.EventsOn(name, callback) ?? (() => {});
}
