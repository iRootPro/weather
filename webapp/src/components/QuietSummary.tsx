import type { QuietSummary as QuietSummaryType } from '../api/dashboard';

export function QuietSummary({ quiet }: { quiet: QuietSummaryType }) {
  if (!quiet.items.length) return null;

  return (
    <section className="quiet-summary">
      <div>
        <span className="quiet-mark">✓</span>
        <h2>{quiet.title}</h2>
      </div>
      <p>{quiet.items.join(' · ')}</p>
    </section>
  );
}
