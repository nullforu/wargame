import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { formatApiError, formatDateTime } from '../lib/utils'
import { buildChartModel, chartLayout, chartUserLimit, type ChartSubmissionPoint, type ChartModel } from '../routes/scoreboardChart'
import type { TimelineResponse } from '../lib/types'
import { navigate } from '../lib/router'
import { getLocaleTag, useLocale, useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'

interface ScoreboardTimelineProps {
    refreshTrigger?: number
}

interface TooltipState {
    left: number
    top: number
    submission: {
        timestamp: string
        user_id: number
        username: string
        points: number
        challenge_count: number
    }
    username: string
}

const ScoreboardTimeline = ({ refreshTrigger = 0 }: ScoreboardTimelineProps) => {
    const t = useT()
    const api = useApi()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const [timeline, setTimeline] = useState<TimelineResponse | null>(null)
    const [chartModel, setChartModel] = useState<ChartModel | null>(null)
    const [hoveredUserId, setHoveredUserId] = useState<number | null>(null)
    const [tooltip, setTooltip] = useState<TooltipState | null>(null)
    const [chartWidth, setChartWidth] = useState(chartLayout.width)
    const chartContainerRef = useRef<HTMLDivElement | null>(null)
    const tooltipBoxRef = useRef<HTMLDivElement | null>(null)
    const requestIdRef = useRef(0)
    const lastModeRef = useRef<'users'>('users')
    const resizeObserverRef = useRef<ResizeObserver | null>(null)
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')

    const showTooltip = useCallback((event: React.MouseEvent, point: ChartSubmissionPoint, username: string) => {
        const chartContainer = chartContainerRef.current
        const tooltipBox = tooltipBoxRef.current
        if (!chartContainer || !tooltipBox) return

        const rect = chartContainer.getBoundingClientRect()
        const tooltipWidth = tooltipBox.offsetWidth
        const tooltipHeight = tooltipBox.offsetHeight
        const padding = 12

        const rawLeft = event.clientX - rect.left + padding
        const rawTop = event.clientY - rect.top + padding
        const maxLeft = rect.width - tooltipWidth - padding
        const maxTop = rect.height - tooltipHeight - padding

        setTooltip({
            left: Math.max(padding, Math.min(rawLeft, maxLeft)),
            top: Math.max(padding, Math.min(rawTop, maxTop)),
            submission: point.submission,
            username,
        })
    }, [])

    const clearTooltip = useCallback(() => {
        setTooltip(null)
    }, [])

    const syncChartSize = useCallback(() => {
        const container = chartContainerRef.current
        if (!container) return
        const nextWidth = Math.floor(container.clientWidth || chartLayout.width)
        setChartWidth((current) => (current !== nextWidth ? nextWidth : current))
    }, [])

    useEffect(() => {
        const container = chartContainerRef.current
        if (!container) return

        syncChartSize()

        if (!resizeObserverRef.current && typeof ResizeObserver !== 'undefined') {
            resizeObserverRef.current = new ResizeObserver(syncChartSize)
            resizeObserverRef.current.observe(container)
        }

        return () => {
            resizeObserverRef.current?.disconnect()
            resizeObserverRef.current = null
        }
    }, [chartModel, syncChartSize])

    useEffect(() => {
        let active = true
        requestIdRef.current += 1
        const currentRequest = requestIdRef.current
        const modeChanged = lastModeRef.current !== 'users'
        lastModeRef.current = 'users'
        setLoading(modeChanged || timeline === null)
        setErrorMessage('')
        if (modeChanged) {
            setChartModel(null)
            setTooltip(null)
        }

        const loadTimeline = async () => {
            try {
                const payload = await api.timeline()
                if (!active || currentRequest !== requestIdRef.current) return
                setTimeline(payload)
            } catch (error) {
                if (active && currentRequest === requestIdRef.current) {
                    setErrorMessage(formatApiError(error, t).message)
                }
            } finally {
                if (active && currentRequest === requestIdRef.current) {
                    setLoading(false)
                }
            }
        }

        loadTimeline()

        return () => {
            active = false
        }
    }, [api, refreshTrigger, t])

    useEffect(() => {
        if (timeline) {
            setChartModel(buildChartModel(timeline, chartWidth, localeTag))
        } else {
            setChartModel(null)
        }
    }, [timeline, chartWidth, localeTag])

    const seriesCount = useMemo(() => chartModel?.series?.length || 0, [chartModel])

    return (
        <div className='min-w-0 rounded-xl border border-border bg-surface p-4'>
            {loading ? (
                <p className='text-sm text-text-muted'>{t('timeline.calculating')}</p>
            ) : errorMessage ? (
                <p className='text-sm text-danger'>{errorMessage}</p>
            ) : timeline ? (
                <>
                    <div className='flex flex-wrap items-center gap-2 text-xs text-text-muted'>
                        <span>{t('timeline.topUsers', { count: Math.min(chartUserLimit, seriesCount) })}</span>
                    </div>
                    {chartModel ? (
                        <div className='mt-4'>
                            <div
                                className='relative min-w-0 w-full overflow-hidden'
                                ref={chartContainerRef}
                                role='group'
                                aria-label={t('timeline.ariaLabel')}
                                onMouseLeave={() => {
                                    setHoveredUserId(null)
                                    clearTooltip()
                                }}
                            >
                                <div className='overflow-x-auto overflow-y-hidden overscroll-x-contain touch-pan-x'>
                                    <div className='w-full' style={{ width: `${chartModel.width}px` }}>
                                        <svg className='block h-72 w-full' viewBox={`0 0 ${chartModel.width} ${chartModel.height}`} role='img' aria-label={t('timeline.ariaLabel')}>
                                            <rect x='0' y='0' width={chartModel.width} height={chartModel.height} fill='transparent' />
                                            <g>
                                                {chartModel.ticks.map((tick) => (
                                                    <g key={`tick-${tick.value}`}>
                                                        <line x1={chartModel.padding.left} x2={chartModel.width - chartModel.padding.right} y1={tick.y} y2={tick.y} className='stroke-border' strokeWidth='1' />
                                                        <text x={chartModel.padding.left - 10} y={tick.y + 4} textAnchor='end' fill='currentColor' style={{ fontSize: 10 }} className='text-text-subtle'>
                                                            {tick.value}
                                                        </text>
                                                    </g>
                                                ))}
                                            </g>
                                            <g>
                                                {chartModel.timeTicks.map((tick) => (
                                                    <g key={`time-${tick.label}-${tick.x}`}>
                                                        <line x1={tick.x} x2={tick.x} y1={chartModel.height - chartModel.padding.bottom} y2={chartModel.height - chartModel.padding.bottom + 6} className='stroke-border' strokeWidth='1' />
                                                        <text x={tick.x} y={chartModel.height - chartModel.padding.bottom + 18} textAnchor='middle' fill='currentColor' style={{ fontSize: 10 }} className='text-text-subtle'>
                                                            {tick.label}
                                                        </text>
                                                    </g>
                                                ))}
                                            </g>
                                            {chartModel.series.map((series) => (
                                                <path
                                                    key={`path-${series.user_id}`}
                                                    d={series.path}
                                                    fill='none'
                                                    stroke={series.color}
                                                    strokeWidth={hoveredUserId === series.user_id ? 3 : 2}
                                                    strokeLinecap='round'
                                                    strokeLinejoin='round'
                                                    className={hoveredUserId && hoveredUserId !== series.user_id ? 'opacity-30' : ''}
                                                    role='presentation'
                                                    aria-hidden='true'
                                                    onMouseEnter={() => setHoveredUserId(series.user_id)}
                                                    onMouseLeave={() => setHoveredUserId(null)}
                                                />
                                            ))}
                                            {chartModel.series.map((series) =>
                                                series.submissionPoints.map((point) => (
                                                    <circle
                                                        key={`point-${series.user_id}-${point.x}-${point.y}`}
                                                        cx={point.x}
                                                        cy={point.y}
                                                        r={hoveredUserId === series.user_id ? 5.5 : 4}
                                                        fill={series.color}
                                                        stroke='currentColor'
                                                        strokeWidth='1.4'
                                                        className={`text-contrast-foreground ${hoveredUserId && hoveredUserId !== series.user_id ? 'opacity-30' : ''}`}
                                                        role='presentation'
                                                        aria-hidden='true'
                                                        onMouseEnter={(event) => {
                                                            setHoveredUserId(series.user_id)
                                                            showTooltip(event, point, series.username)
                                                        }}
                                                        onMouseMove={(event) => showTooltip(event, point, series.username)}
                                                        onMouseLeave={() => {
                                                            setHoveredUserId(null)
                                                            clearTooltip()
                                                        }}
                                                    />
                                                )),
                                            )}
                                        </svg>
                                    </div>
                                </div>
                                <div
                                    className={`pointer-events-none absolute z-10 w-60 max-w-[70vw] rounded-lg border border-border bg-surface/95 p-3 text-xs text-text shadow-xl ${tooltip ? '' : 'hidden'}`}
                                    ref={tooltipBoxRef}
                                    style={{ left: `${tooltip?.left ?? 0}px`, top: `${tooltip?.top ?? 0}px` }}
                                >
                                    {tooltip ? (
                                        <>
                                            <p className='text-text'>{t('timeline.tooltipUser', { name: tooltip.username })}</p>
                                            <p className='mt-1 text-sm text-text'>
                                                {tooltip.submission.challenge_count > 1
                                                    ? t('timeline.tooltipSolvedMany', {
                                                          count: tooltip.submission.challenge_count,
                                                      })
                                                    : t('timeline.tooltipSolvedOne')}
                                            </p>
                                            <p className='mt-1 text-text-muted'>{formatDateTime(tooltip.submission.timestamp, localeTag)}</p>
                                            <p className='mt-1 text-accent'>{t('timeline.tooltipPoints', { points: tooltip.submission.points })}</p>
                                        </>
                                    ) : null}
                                </div>
                            </div>
                            <div className='mt-3 flex flex-wrap gap-3 text-xs text-text-muted'>
                                {chartModel.series.map((series) => (
                                    <button
                                        key={`legend-${series.user_id}`}
                                        className={`flex items-center gap-2 transition ${hoveredUserId && hoveredUserId !== series.user_id ? 'opacity-40' : ''} ${hoveredUserId === series.user_id ? 'text-text' : ''} cursor-pointer`}
                                        aria-label={t('timeline.highlight', { name: series.username })}
                                        onMouseEnter={() => setHoveredUserId(series.user_id)}
                                        onMouseLeave={() => setHoveredUserId(null)}
                                        onClick={() => navigate(`/users/${series.user_id}`)}
                                    >
                                        <span className='h-2 w-2 rounded-full' style={{ backgroundColor: series.color }}></span>
                                        {series.username}
                                    </button>
                                ))}
                            </div>
                            <div className='mt-2 flex justify-between text-xs text-text-muted'>
                                <span>{formatDateTime(chartModel.startLabel, localeTag)}</span>
                                <span>{formatDateTime(chartModel.endLabel, localeTag)}</span>
                            </div>
                        </div>
                    ) : (
                        <p className='mt-4 text-sm text-text-muted'>{t('timeline.noData')}</p>
                    )}
                </>
            ) : null}
        </div>
    )
}

export default ScoreboardTimeline
