// Service states are specific. Offline means the service answered and routing is stopped.
export function runStateLabel(
  status: { running?: boolean; starting?: boolean; service?: string } | null,
  starting: boolean,
): string {
  if (starting || status?.starting) return "Starting…";
  switch (status?.service) {
    case "uninstalled":
      return "Service not installed";
    case "authorization":
      return "Authorization required";
    case "stopped":
      return "Engine stopped";
    case "unreachable":
      return "Communication error";
    default:
      return status?.running ? "Online" : "Offline";
  }
}
