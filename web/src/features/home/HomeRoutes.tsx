import { useMatch } from 'react-router-dom'
import { Shell } from '../../components/Shell'
import { DetailView } from '../detail/DetailView'
import { Panel } from '../detail/Panel'
import { useMediaQuery } from '../../lib/useMediaQuery'
import { HomePage } from './HomePage'
import { useDashboard } from './useDashboard'

/** Masaüstünde detay ana ekranın üstünde panel olarak, telefonda ayrı sayfa olarak açılır. */
export function HomeRoutes() {
  const id = useMatch('/m/:id')?.params.id
  const wide = useMediaQuery('(min-width: 900px)')
  const dashboard = useDashboard()

  if (id && !wide) {
    return (
      <Shell maxWidth={720}>
        <DetailView key={id} id={id} onChanged={dashboard.reload} />
      </Shell>
    )
  }
  return (
    <>
      <HomePage dashboard={dashboard} />
      {id && (
        <Panel key={id} labelledBy="detail-title">
          {(close) => <DetailView id={id} onChanged={dashboard.reload} onClose={close} />}
        </Panel>
      )}
    </>
  )
}
