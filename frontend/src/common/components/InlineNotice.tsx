export function InlineNotice({
  tone,
  title,
  detail,
}: {
  tone: 'error' | 'warn' | 'info'
  title: string
  detail?: string
}) {
  return (
    <section className={`inline-notice ${tone}`}>
      <strong>{title}</strong>
      {detail ? <p>{detail}</p> : null}
    </section>
  )
}
