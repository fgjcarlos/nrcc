export function InlineNotice({
  tone,
  title,
  detail,
}: {
  tone: 'error' | 'warn' | 'info'
  title: string
  detail?: string
}) {
  const getToneClass = () => {
    switch (tone) {
      case 'error':
        return 'alert-error'
      case 'warn':
        return 'alert-warning'
      case 'info':
      default:
        return 'alert-info'
    }
  }

  const getIconEmoji = () => {
    switch (tone) {
      case 'error':
        return '❌'
      case 'warn':
        return '⚠️'
      case 'info':
      default:
        return 'ℹ️'
    }
  }

  return (
    <section className={`alert ${getToneClass()} shadow-md`}>
      <div className="flex items-start gap-3 w-full">
        <span className="text-lg flex-shrink-0">{getIconEmoji()}</span>
        <div className="flex-1">
          <strong className="text-sm">{title}</strong>
          {detail ? <p className="text-xs opacity-90 mt-1">{detail}</p> : null}
        </div>
      </div>
    </section>
  )
}
