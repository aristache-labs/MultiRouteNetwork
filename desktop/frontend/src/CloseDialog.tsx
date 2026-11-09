import { useEffect, useRef, useState } from "react";
import { api } from "./api";

export function CloseDialog() {
  const dialog = useRef<HTMLDialogElement>(null);
  const cancelButton = useRef<HTMLButtonElement>(null);
  const [requested, setRequested] = useState(false);
  const [remember, setRemember] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const request = (event: Event) => {
      setError((event as CustomEvent<string>).detail || "");
      setRemember(false);
      setRequested(true);
    };
    window.addEventListener("multiroute:close", request);
    // The native close request is durable. Poll it as well as listening for the
    // immediate notification: a lost WebView event must not strand the X button.
    let active = true;
    let inFlight = false;
    const checkPending = async () => {
      if (inFlight) return;
      inFlight = true;
      try {
        const p = await api.windowPreferences();
        if (active && p.pending) {
          setError(p.closeError || "");
          setRequested(true);
        }
      } catch {
        /* A later poll retries the local window endpoint. */
      } finally {
        inFlight = false;
      }
    };
    void checkPending();
    const timer = window.setInterval(() => void checkPending(), 500);
    return () => {
      active = false;
      window.clearInterval(timer);
      window.removeEventListener("multiroute:close", request);
    };
  }, []);

  useEffect(() => {
    if (requested && !dialog.current?.open) {
      dialog.current?.showModal();
      cancelButton.current?.focus();
    } else if (!requested && dialog.current?.open) dialog.current.close();
  }, [requested]);

  async function choose(action: "tray" | "exit" | "cancel") {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await api.closeWindow(action, remember);
      setRequested(false);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Could not close the window. Try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <dialog
      ref={dialog}
      className="close-dialog"
      aria-labelledby="close-dialog-title"
      aria-describedby="close-dialog-description"
      onCancel={(e) => {
        e.preventDefault();
        if (!busy) void choose("cancel");
      }}
    >
      <header className="close-dialog-heading">
        <span className="close-dialog-mark" aria-hidden="true">
          M
        </span>
        <button
          className="icon-button"
          aria-label="Dismiss close dialog"
          disabled={busy}
          onClick={() => void choose("cancel")}
        >
          <svg
            className="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
          >
            <path d="m5 5 14 14M5 19 19 5" />
          </svg>
        </button>
      </header>
      <h2 id="close-dialog-title">Close MultiRoute?</h2>
      <p id="close-dialog-description" className="muted">
        Would you like to hide the window or exit the app?
      </p>
      <div className="close-options">
        <button
          className="close-option"
          disabled={busy}
          onClick={() => void choose("tray")}
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            className="icon"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.7"
          >
            <rect x="3" y="3" width="18" height="13" rx="2" />
            <path d="M8 21h8m-4-9v9m-3-3 3 3 3-3" />
          </svg>
          <span>
            <strong>Hide in system tray</strong>
            <small>Keep MultiRoute running in the background.</small>
          </span>
        </button>
        <button
          className="close-option"
          disabled={busy}
          onClick={() => void choose("exit")}
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            className="icon"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.7"
          >
            <path d="M12 2v10M6 5a9 9 0 1 0 12 0" />
          </svg>
          <span>
            <strong>Exit MultiRoute</strong>
            <small>Stop routing and close the app completely.</small>
          </span>
        </button>
      </div>
      <label className="remember-close">
        <input
          type="checkbox"
          checked={remember}
          disabled={busy}
          onChange={(e) => setRemember(e.target.checked)}
        />
        <span>Remember my choice and don’t ask again</span>
      </label>
      <p className="close-preference-hint muted">
        You can change this later in Settings.
      </p>
      {error && (
        <p className="close-error" role="alert">
          {error}
        </p>
      )}
      <footer className="close-dialog-footer">
        <button
          ref={cancelButton}
          disabled={busy}
          onClick={() => void choose("cancel")}
        >
          Cancel
        </button>
      </footer>
    </dialog>
  );
}

export function CloseBehaviorSettings() {
  const [action, setAction] = useState<"ask" | "tray" | "exit">("ask");
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    api
      .windowPreferences()
      .then((p) => {
        if (active) setAction(p.action);
      })
      .catch((err: Error) => {
        if (active) setError(err.message);
      })
      .finally(() => {
        if (active) setBusy(false);
      });
    return () => {
      active = false;
    };
  }, []);
  async function save(next: "ask" | "tray" | "exit") {
    setBusy(true);
    setError("");
    try {
      await api.saveClosePreference(next);
      setAction(next);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Could not save the close preference.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="field-label">
      <label htmlFor="window-close-behavior">When closing the window</label>
      <select
        id="window-close-behavior"
        value={action}
        disabled={busy}
        onChange={(e) => void save(e.target.value as "ask" | "tray" | "exit")}
      >
        <option value="ask">Ask every time</option>
        <option value="tray">Hide in system tray</option>
        <option value="exit">Exit MultiRoute</option>
      </select>
      {error && (
        <span role="alert" className="close-error">
          {error}
        </span>
      )}
    </div>
  );
}
