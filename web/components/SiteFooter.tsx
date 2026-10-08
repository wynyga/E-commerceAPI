export default function SiteFooter() {
  return (
    <footer className="mt-16 border-t border-line">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-6 text-sm text-muted sm:px-6">
        <p>Kedai Seduh is a demo storefront for a Go microservice backend. No real orders, no payments.</p>
        <a className="underline hover:text-ink" href="https://github.com/wynyga/E-commerceAPI" target="_blank" rel="noreferrer">
          Source on GitHub
        </a>
      </div>
    </footer>
  );
}
