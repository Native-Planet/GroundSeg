export const daysUntilDate = dateString => {
  const targetDate = new Date(dateString)
  const currentDate = new Date()
  const diffTime = targetDate - currentDate
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))
  return Math.max(diffDays, 0)
}
