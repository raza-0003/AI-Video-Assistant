const LABELS = {
  pending: "Pending",
  processing: "Processing",
  completed: "Completed",
  failed: "Failed",
  cancelled: "Cancelled",
};

export default function StatusBadge({ status }) {
  return (
    <span className={`badge ${status}`}>
      <span className="badge-dot" />
      {LABELS[status] || status}
    </span>
  );
}
