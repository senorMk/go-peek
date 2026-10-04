export type StreamEvent = { requestId: string; kind: 'text' | 'done' | 'cancelled'; text?: string };
type Desktop = {
  GetStatus(): Promise<{ captureProtectionRequested: boolean }>;
  StartDemo(marker: string): Promise<string>;
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
