export function Logo({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="currentColor"
        d="M5.2 2 7.4 4Q8.6 3.6 9.8 4L11.6 2.2 11.8 6C11.9 7.4 11.4 8.4 10.6 9C13.6 9.8 16.8 12.6 17.6 16.6C18.2 19.6 17.4 21.6 15.6 22.4H8.2C7.6 22.4 7.4 21.8 7.8 21.4C8.6 20.6 8.6 17.6 8.2 14.2C7.9 11.8 7 10.6 6.2 9.6C4.8 8.8 4.4 7.4 4.6 6Z"
      />
      <path
        d="M15.4 22.4C19.4 22.6 21.6 21 21.4 17.6C21.3 15.6 20.2 14.4 19 14.6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
    </svg>
  )
}
