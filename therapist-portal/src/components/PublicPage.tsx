import pages from '../content/public-pages.json';
import { SiteFrame, SiteHeader, SiteFooter } from './SiteFrame';
export default function PublicPage({
  page
}: {
  page: keyof typeof pages;
}) {
  // Repository-owned static templates only. API and user data use React rendering.
  return <SiteFrame><a className="skip" href="#main">Skip to content</a><SiteHeader /><main id="main" tabIndex={-1} dangerouslySetInnerHTML={{
      __html: pages[page]
    }} /><SiteFooter /></SiteFrame>;
}
