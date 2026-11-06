import { useEffect, useLayoutEffect, useRef, useState, type MouseEvent, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import styles from './Panel.module.css'

const CLOSE_MS = 220

interface Props {
  labelledBy: string
  children: (close: () => void) => ReactNode
}

/** Yerel dialog odağı kilitler, arkayı etkisizleştirir ve kapanınca odağı tetikleyene geri verir. */
export function Panel({ labelledBy, children }: Props) {
  const navigate = useNavigate()
  const dialog = useRef<HTMLDialogElement>(null)
  const [closing, setClosing] = useState(false)

  // Layout effect: close() odağı ancak dialog henüz DOM'dayken geri verebilir.
  useLayoutEffect(() => {
    const element = dialog.current!
    element.showModal()
    element.focus()
    return () => element.close()
  }, [])

  useEffect(() => {
    if (!closing) return
    const timer = setTimeout(() => navigate('/'), CLOSE_MS)
    return () => clearTimeout(timer)
  }, [closing, navigate])

  const close = () => setClosing(true)

  function onClick(event: MouseEvent<HTMLDialogElement>) {
    // İçerik paneli tamamen kapladığı için dialog'un kendisine gelen tıklama scrim'dendir.
    if (event.target === event.currentTarget) close()
  }

  return (
    <dialog
      ref={dialog}
      tabIndex={-1}
      aria-labelledby={labelledBy}
      className={`${styles.panel} ${closing ? styles.closing : ''}`}
      onClick={onClick}
      onCancel={(event) => {
        event.preventDefault()
        close()
      }}
    >
      <div className={styles.body}>{children(close)}</div>
    </dialog>
  )
}
