import type { AnchorHTMLAttributes, ButtonHTMLAttributes } from 'react'
import { Link, type LinkProps } from 'react-router-dom'
import styles from './Button.module.css'

type Variant = 'primary' | 'secondary' | 'ghost' | 'icon'
type Size = 'xs' | 'sm' | 'md'

interface Look {
  variant?: Variant
  size?: Size
}

function classes({ variant = 'secondary', size = 'sm' }: Look, className?: string) {
  return [styles.button, styles[variant], styles[size], className].filter(Boolean).join(' ')
}

export function Button({
  variant,
  size,
  className,
  type = 'button',
  ...rest
}: Look & ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button type={type} className={classes({ variant, size }, className)} {...rest} />
}

export function ButtonLink({ variant, size, className, ...rest }: Look & LinkProps) {
  return <Link className={classes({ variant, size }, className)} {...rest} />
}

/** Router dışı adresler için, örneğin indirme bağlantıları. */
export function ButtonAnchor({ variant, size, className, ...rest }: Look & AnchorHTMLAttributes<HTMLAnchorElement>) {
  return <a className={classes({ variant, size }, className)} {...rest} />
}
