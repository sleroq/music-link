import SharePage from './SharePage';
import { pageShare } from './share';

export default function App() {
  return <SharePage share={pageShare()} />;
}
