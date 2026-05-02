import { useCallback, useEffect, useMemo, useState } from 'react'
import useEmblaCarousel from 'embla-carousel-react'
import UserAvatar from '../components/UserAvatar'
import { useAuth } from '../lib/auth'
import { getLocaleTag, useLocale, useT } from '../lib/i18n'
import { navigate } from '../lib/router'
import type { AffiliationRankingEntry, AuthUser, Challenge, CommunityPost, UserRankingEntry } from '../lib/types'
import { useApi } from '../lib/useApi'
import { formatApiError, formatDateTime } from '../lib/utils'
import { categoryBadgeClass, categoryTextKey } from './Community'
import { LevelBadge } from './Challenges'
import { SITE_CONFIG } from '../lib/siteConfig'

interface RouteProps {
    routeParams?: Record<string, string>
}

const POPULAR_POST_LIKE_THRESHOLD = 5
const LaunchIcon = () => (
    <svg viewBox='0 0 24 24' className='h-4 w-4' fill='none' stroke='currentColor' strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round' aria-hidden='true'>
        <path d='M7 17 17 7' />
        <path d='M9 7h8v8' />
    </svg>
)

const Home = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const api = useApi()
    const { state: auth } = useAuth()

    const [profile, setProfile] = useState<AuthUser | null>(null)
    const [challengeRows, setChallengeRows] = useState<Challenge[]>([])
    const [noticeRows, setNoticeRows] = useState<CommunityPost[]>([])
    const [communityRows, setCommunityRows] = useState<CommunityPost[]>([])
    const [rankingRows, setRankingRows] = useState<UserRankingEntry[]>([])
    const [affiliationRankingRows, setAffiliationRankingRows] = useState<AffiliationRankingEntry[]>([])
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')
    const [emblaRef, emblaApi] = useEmblaCarousel({ align: 'start', containScroll: 'trimSnaps', dragFree: false, slidesToScroll: 1 })
    const [canScrollPrev, setCanScrollPrev] = useState(false)
    const [canScrollNext, setCanScrollNext] = useState(false)
    const displayProfile = profile ?? auth.user

    useEffect(() => {
        let active = true
        setLoading(true)
        setErrorMessage('')

        const load = async () => {
            try {
                const baseRequests: Array<Promise<any>> = [
                    api.challenges(1, 10),
                    api.communityPosts({ page: 1, pageSize: 5, category: 0, sort: 'latest' }),
                    api.communityPosts({ page: 1, pageSize: 10, excludeNotice: true, sort: 'latest' }),
                    api.rankingUsers(1, 10),
                    api.rankingAffiliations(1, 5),
                ]

                const results = auth.user ? await Promise.all([api.me(), ...baseRequests]) : await Promise.all(baseRequests)
                if (!active) return

                if (auth.user) {
                    setProfile(results[0] as AuthUser)
                    setChallengeRows((results[1] as { challenges: Challenge[] }).challenges)
                    setNoticeRows((results[2] as { posts: CommunityPost[] }).posts)
                    setCommunityRows((results[3] as { posts: CommunityPost[] }).posts)
                    setRankingRows((results[4] as { entries: UserRankingEntry[] }).entries)
                    setAffiliationRankingRows((results[5] as { entries: AffiliationRankingEntry[] }).entries)
                } else {
                    setProfile(null)
                    setChallengeRows((results[0] as { challenges: Challenge[] }).challenges)
                    setNoticeRows((results[1] as { posts: CommunityPost[] }).posts)
                    setCommunityRows((results[2] as { posts: CommunityPost[] }).posts)
                    setRankingRows((results[3] as { entries: UserRankingEntry[] }).entries)
                    setAffiliationRankingRows((results[4] as { entries: AffiliationRankingEntry[] }).entries)
                }
            } catch (error) {
                if (!active) return
                setErrorMessage(formatApiError(error, t).message)
            } finally {
                if (active) setLoading(false)
            }
        }

        void load()
        return () => {
            active = false
        }
    }, [api, auth.user, t])

    const updateEmblaButtons = useCallback(() => {
        if (!emblaApi) return
        setCanScrollPrev(emblaApi.canScrollPrev())
        setCanScrollNext(emblaApi.canScrollNext())
    }, [emblaApi])

    useEffect(() => {
        if (!emblaApi) return
        updateEmblaButtons()
        emblaApi.on('select', updateEmblaButtons)
        emblaApi.on('reInit', updateEmblaButtons)
        return () => {
            emblaApi.off('select', updateEmblaButtons)
            emblaApi.off('reInit', updateEmblaButtons)
        }
    }, [emblaApi, updateEmblaButtons])

    return (
        <section className='animate space-y-12'>
            {errorMessage ? <p className='text-sm text-danger'>{errorMessage}</p> : null}

            <div className='relative -mt-5 ml-[calc(50%-50vw)] mr-[calc(50%-50vw)] w-screen overflow-hidden bg-linear-to-br from-accent/16 via-surface-muted to-info/10 py-10 md:-mt-6 md:py-14 mb-6'>
                {SITE_CONFIG.homeBannerImage ? (
                    <div className='pointer-events-none absolute inset-0'>
                        <img src={SITE_CONFIG.homeBannerImage} alt='' className={`h-full w-full ${SITE_CONFIG.homeBannerImageFit === 'contain' ? 'object-contain' : 'object-cover'} opacity-30`} />
                        <div className='absolute inset-0 bg-surface/55 dark:bg-surface/65' />
                    </div>
                ) : null}
                <div className='relative mx-auto w-full max-w-7xl px-6 md:px-8'>
                    <div className='flex flex-wrap items-start justify-between gap-4'>
                        <div>
                            <p className='text-xs font-semibold tracking-[0.09em] text-accent'>{t('home.bannerEyebrow')}</p>
                            <h1 className='mt-2 text-3xl font-semibold text-text md:text-4xl'>{t('home.bannerTitle')}</h1>
                            <p className='mt-2 text-sm text-text-muted'>{t('home.bannerBody')}</p>
                        </div>
                    </div>
                </div>
            </div>

            <div className='grid gap-10 xl:grid-cols-[0.95fr_1.55fr]'>
                <section>
                    {loading ? (
                        <div className='rounded-xl border border-border/70 p-4'>
                            <div className='flex items-start justify-between gap-3'>
                                <div className='flex items-center gap-3'>
                                    <div className='h-12 w-12 animate-pulse rounded-full bg-surface-muted' />
                                    <div className='space-y-2'>
                                        <div className='h-4 w-32 animate-pulse rounded bg-surface-muted' />
                                        <div className='h-3 w-40 animate-pulse rounded bg-surface-muted' />
                                    </div>
                                </div>
                                <div className='h-8 w-22 animate-pulse rounded-md bg-surface-muted' />
                            </div>
                            <div className='mt-3 h-4 w-full animate-pulse rounded bg-surface-muted' />
                            <div className='mt-1 h-4 w-3/4 animate-pulse rounded bg-surface-muted' />
                            <div className='mt-4 grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-2'>
                                {Array.from({ length: 6 }, (_, idx) => (
                                    <div key={`profile-chip-skeleton-${idx}`} className='h-14 animate-pulse rounded-md bg-surface-muted' />
                                ))}
                            </div>
                        </div>
                    ) : displayProfile ? (
                        <div className='rounded-xl border border-border/70 p-4'>
                            <div className='flex items-start justify-between gap-3'>
                                <div className='flex min-w-0 items-center gap-3'>
                                    <UserAvatar username={displayProfile.username} size='md' />
                                    <div className='min-w-0'>
                                        <p className='truncate text-lg font-semibold text-text'>{displayProfile.username}</p>
                                        <p className='truncate text-xs text-text-subtle'>{displayProfile.affiliation?.trim() ? displayProfile.affiliation : t('common.na')}</p>
                                    </div>
                                </div>
                                <button
                                    type='button'
                                    className='inline-flex shrink-0 items-center gap-1 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium text-text-muted transition hover:border-accent/35 hover:text-accent'
                                    onClick={() => navigate('/profile')}
                                    aria-label={t('profile.title')}
                                >
                                    <span>{t('profile.title')}</span>
                                    <LaunchIcon />
                                </button>
                            </div>
                            <p className='mt-3 line-clamp-2 rounded-md px-1 py-1.5 text-sm text-text-muted'>{displayProfile.bio ?? t('profile.noBio')}</p>
                            <div className='mt-4 grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-2'>
                                <div className='rounded-md border border-border/50 px-3 py-2'>
                                    <p className='text-[11px] text-text-subtle'>{t('common.email')}</p>
                                    <p className='truncate text-sm font-semibold text-text'>{displayProfile.email}</p>
                                </div>
                                <div className='rounded-md border border-border/50 px-3 py-2'>
                                    <p className='text-[11px] text-text-subtle'>{t('common.affiliation')}</p>
                                    <p className='truncate text-sm font-semibold text-text'>{displayProfile.affiliation?.trim() ? displayProfile.affiliation : t('profile.noAffiliation')}</p>
                                </div>
                                <div className='rounded-md border border-border/50 px-3 py-2'>
                                    <p className='text-[11px] text-text-subtle'>Stack</p>
                                    <p className='text-sm font-semibold text-text'>{displayProfile.stack_count}</p>
                                </div>
                                <div className='rounded-md border border-border/50 px-3 py-2'>
                                    <p className='text-[11px] text-text-subtle'>{t('home.stackLimitLabel')}</p>
                                    <p className='text-sm font-semibold text-text'>{displayProfile.stack_limit}</p>
                                </div>
                            </div>
                        </div>
                    ) : (
                        <div className='rounded-xl border border-border/50 p-4'>
                            <p className='text-sm text-text-muted'>{t('home.profileGuestMessage')}</p>
                        </div>
                    )}
                </section>

                <section>
                    <div className='mb-3 flex items-end justify-between gap-2'>
                        <a href='/community?category=0' onClick={(e) => navigate('/community?category=0', e)} className='inline-flex items-center gap-1 text-xl font-semibold text-text transition hover:text-accent md:text-2xl'>
                            <span>{t('home.recentNoticesTitle')}</span>
                            <LaunchIcon />
                        </a>
                    </div>
                    <div className='relative pb-5'>
                        <div className='divide-y divide-border/70 border-y border-border/70'>
                            {loading
                                ? Array.from({ length: 5 }, (_, idx) => (
                                      <div key={`notice-skeleton-${idx}`} className='grid grid-cols-[1fr_auto] gap-3 px-1 py-3'>
                                          <div className='h-4 w-full animate-pulse rounded bg-surface-muted' />
                                          <div className='h-4 w-20 animate-pulse rounded bg-surface-muted' />
                                      </div>
                                  ))
                                : noticeRows.map((post) => (
                                      <button
                                          key={`notice-${post.id}`}
                                          type='button'
                                          className='grid w-full grid-cols-[1fr_auto] gap-3 px-1 py-3 text-left transition hover:bg-surface-muted/70'
                                          onClick={() => navigate(`/community/${post.id}`)}
                                      >
                                          <p className='line-clamp-1 text-sm text-text'>{post.title}</p>
                                          <span className='text-xs text-text-subtle'>{formatDateTime(post.created_at, localeTag)}</span>
                                      </button>
                                  ))}
                            {!loading && noticeRows.length === 0 ? <p className='px-1 py-4 text-sm text-text-subtle'>{t('home.noNotices')}</p> : null}
                        </div>
                        <span className='absolute bottom-0 right-0 text-[11px] text-text-subtle'>{t('home.latestCountHint', { count: 5 })}</span>
                    </div>
                </section>
            </div>

            <section>
                <div className='mb-3 flex items-end justify-between gap-2'>
                    <div className='space-y-1'>
                        <a href='/challenges' onClick={(e) => navigate('/challenges', e)} className='inline-flex items-center gap-1 text-xl font-semibold text-text transition hover:text-accent md:text-2xl'>
                            <span>{t('home.latestChallengesTitle')}</span>
                            <LaunchIcon />
                        </a>
                    </div>
                    <div className='flex items-center gap-2 self-end'>
                        <button
                            type='button'
                            className='rounded-md border border-border bg-surface px-2 py-1 text-xs text-text transition hover:bg-surface-muted disabled:opacity-40'
                            onClick={() => emblaApi?.scrollPrev()}
                            disabled={!canScrollPrev}
                            aria-label={t('common.previous')}
                        >
                            {'<'}
                        </button>
                        <button
                            type='button'
                            className='rounded-md border border-border bg-surface px-2 py-1 text-xs text-text transition hover:bg-surface-muted disabled:opacity-40'
                            onClick={() => emblaApi?.scrollNext()}
                            disabled={!canScrollNext}
                            aria-label={t('common.next')}
                        >
                            {'>'}
                        </button>
                    </div>
                </div>
                <div className='relative pb-5'>
                    {loading ? (
                        <div className='flex gap-3 overflow-x-auto pb-1'>
                            {Array.from({ length: 8 }, (_, idx) => (
                                <div key={`challenge-card-skeleton-${idx}`} className='h-72 min-w-44 shrink-0 animate-pulse rounded-xl border border-border bg-surface-muted' />
                            ))}
                        </div>
                    ) : challengeRows.length === 0 ? (
                        <p className='px-1 py-4 text-sm text-text-subtle'>{t('home.noChallenges')}</p>
                    ) : (
                        <div className='overflow-hidden mb-2' ref={emblaRef}>
                            <div className='flex gap-4'>
                                {challengeRows.map((challenge) => (
                                    <div key={`challenge-card-${challenge.id}`} className='min-w-0 flex-[0_0_65%] sm:flex-[0_0_38%] lg:flex-[0_0_23%] xl:flex-[0_0_17%]'>
                                        <button
                                            type='button'
                                            className='group h-72 w-full rounded-xl border border-border/50 text-left transition hover:-translate-y-0.5 hover:border-accent/45 hover:bg-surface-muted'
                                            onClick={() => navigate(`/challenges/${challenge.id}`)}
                                        >
                                            <div className='flex h-full flex-col'>
                                                <div className='flex items-center gap-2 border-b border-border/50 px-4 py-3'>
                                                    <UserAvatar username={challenge.created_by?.username ?? 'unknown'} size='sm' />
                                                    <div className='min-w-0'>
                                                        <p className='truncate text-sm font-semibold text-text'>{challenge.created_by?.username ?? t('common.na')}</p>
                                                        <p className='truncate text-[11px] text-text-subtle'>{challenge.created_by?.affiliation ?? t('common.na')}</p>
                                                    </div>
                                                </div>
                                                <div className='flex flex-1 flex-col items-center justify-center px-3 py-2'>
                                                    <div className='scale-[1.7] md:scale-[1.85]'>
                                                        <LevelBadge level={challenge.level} />
                                                    </div>
                                                    <p className='mt-6 line-clamp-2 text-center text-base font-semibold text-text break-keep'>{challenge.title}</p>
                                                    <p className='mt-1 text-center text-xs text-text-subtle'>{challenge.category}</p>
                                                </div>
                                                <div className='border-t border-border/50 px-3 py-2'>
                                                    <p className='text-xs text-text-subtle'>{t('home.challengeMeta', { category: challenge.category, points: challenge.points, solved: challenge.solve_count })}</p>
                                                </div>
                                            </div>
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                    <span className='absolute bottom-0 right-0 text-[11px] text-text-subtle'>{t('home.latestCountHint', { count: 10 })}</span>
                </div>
            </section>

            <div className='grid gap-10 xl:grid-cols-[1.35fr_1fr]'>
                <div className='space-y-8'>
                    <section>
                        <div className='mb-3 flex items-end justify-between gap-2'>
                            <a href='/community' onClick={(e) => navigate('/community', e)} className='inline-flex items-center gap-1 text-xl font-semibold text-text transition hover:text-accent md:text-2xl'>
                                <span>{t('home.recentPostsTitle')}</span>
                                <LaunchIcon />
                            </a>
                        </div>
                        <div className='relative pb-5'>
                            <div className='hidden md:grid grid-cols-[70px_minmax(0,1.2fr)_100px_110px_70px] items-center gap-3 px-4 py-3 text-xs font-medium text-text-muted'>
                                <p>{t('common.category')}</p>
                                <p>{t('common.title')}</p>
                                <p>{t('common.username')}</p>
                                <p>{t('common.createdAt')}</p>
                                <p>{t('community.table.likes')}</p>
                            </div>
                            {loading ? (
                                <div className='divide-y divide-border/60'>
                                    {Array.from({ length: 8 }, (_, idx) => (
                                        <div key={`community-skeleton-${idx}`}>
                                            <div className='hidden md:grid grid-cols-[70px_minmax(0,1.2fr)_100px_110px_70px] items-center gap-3 px-4 py-3'>
                                                <div className='h-7 w-14 animate-pulse rounded-full bg-surface-muted' />
                                                <div className='h-4 w-full animate-pulse rounded bg-surface-muted' />
                                                <div className='h-4 w-16 animate-pulse rounded bg-surface-muted' />
                                                <div className='h-4 w-18 animate-pulse rounded bg-surface-muted' />
                                                <div className='h-4 w-10 animate-pulse rounded bg-surface-muted' />
                                            </div>
                                            <div className='px-4 py-3 md:hidden'>
                                                <div className='h-4 w-20 animate-pulse rounded bg-surface-muted' />
                                                <div className='mt-2 h-5 w-full animate-pulse rounded bg-surface-muted' />
                                                <div className='mt-2 h-4 w-2/3 animate-pulse rounded bg-surface-muted' />
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <div className='divide-y divide-border/60'>
                                    {communityRows.map((post) => (
                                        <div key={post.id}>
                                            <button
                                                className={`rounded-none hidden w-full md:grid grid-cols-[70px_minmax(0,1.2fr)_100px_110px_70px] items-center gap-3 px-4 py-3 text-left transition hover:bg-surface-muted/40 ${post.category === 0 ? 'bg-warning/10 hover:bg-warning/20' : ''}`}
                                                onClick={() => navigate(`/community/${post.id}${window.location.search}`)}
                                            >
                                                <p>
                                                    <span className={`rounded-md inline-flex border px-2 py-0.5 text-[11px] font-medium ${categoryBadgeClass(post.category)}`}>{t(categoryTextKey(post.category))}</span>
                                                </p>
                                                <p className='flex min-w-0 items-center gap-1.5 text-sm font-medium text-text'>
                                                    {post.like_count >= POPULAR_POST_LIKE_THRESHOLD ? (
                                                        <svg viewBox='0 0 24 24' className='h-4 w-4 shrink-0 text-warning' fill='currentColor' aria-label='popular'>
                                                            <path d='M12 2.5l2.9 5.88 6.49.94-4.7 4.58 1.11 6.47L12 17.32l-5.8 3.05 1.1-6.47-4.69-4.58 6.49-.94L12 2.5Z' />
                                                        </svg>
                                                    ) : null}
                                                    <span className='truncate'>{post.title}</span>
                                                    {post.comment_count > 0 ? <span className='shrink-0 text-xs font-semibold text-text-subtle'>({post.comment_count})</span> : null}
                                                </p>
                                                <p className='truncate text-sm text-text-muted'>{post.author.username}</p>
                                                <p className='text-sm text-text-muted'>{formatDateTime(post.created_at, localeTag).slice(0, 11)}</p>
                                                <p className='text-sm text-text-muted'>{post.like_count}</p>
                                            </button>

                                            <button
                                                className={`rounded-none w-full px-4 py-3 text-left md:hidden ${post.category === 0 ? 'bg-warning/10 hover:bg-warning/20' : ''}`}
                                                onClick={() => navigate(`/community/${post.id}${window.location.search}`)}
                                            >
                                                <div className='flex items-center justify-between gap-2'>
                                                    <span className={`rounded-md inline-flex border px-2 py-0.5 text-[11px] font-medium ${categoryBadgeClass(post.category)}`}>{t(categoryTextKey(post.category))}</span>
                                                    <span className='text-xs text-text-subtle'>{formatDateTime(post.created_at, localeTag)}</span>
                                                </div>
                                                <p className='mt-1 flex items-center gap-1.5 text-sm font-semibold text-text'>
                                                    {post.like_count >= POPULAR_POST_LIKE_THRESHOLD ? (
                                                        <svg viewBox='0 0 24 24' className='h-4 w-4 shrink-0 text-warning' fill='currentColor' aria-label='popular'>
                                                            <path d='M12 2.5l2.9 5.88 6.49.94-4.7 4.58 1.11 6.47L12 17.32l-5.8 3.05 1.1-6.47-4.69-4.58 6.49-.94L12 2.5Z' />
                                                        </svg>
                                                    ) : null}
                                                    <span className='line-clamp-1'>{post.title}</span>
                                                    {post.comment_count > 0 ? <span className='shrink-0 text-xs font-semibold text-text-subtle'>({post.comment_count})</span> : null}
                                                </p>
                                                <div className='mt-1 flex items-center gap-2 text-xs text-text-muted'>
                                                    <span>{post.author.username}</span>
                                                    <span>·</span>
                                                    <span>{t('community.likes', { count: post.like_count })}</span>
                                                </div>
                                            </button>
                                        </div>
                                    ))}
                                </div>
                            )}
                            {!loading && communityRows.length === 0 ? <p className='px-4 py-8 text-center text-sm text-text-muted'>{t('community.empty')}</p> : null}
                            <span className='absolute bottom-0 right-2 text-[11px] text-text-subtle'>{t('home.latestCountHint', { count: 10 })}</span>
                        </div>
                    </section>
                </div>

                <div className='space-y-8'>
                    <section>
                        <div className='mb-3 flex items-end justify-between gap-2'>
                            <a href='/ranking' onClick={(e) => navigate('/ranking', e)} className='inline-flex items-center gap-1 text-xl font-semibold text-text transition hover:text-accent md:text-2xl'>
                                <span>{t('home.topUsersTitle')}</span>
                                <LaunchIcon />
                            </a>
                        </div>
                        <div className='relative space-y-1 pb-5'>
                            {loading
                                ? Array.from({ length: 10 }, (_, idx) => (
                                      <div key={`ranking-skeleton-${idx}`} className='flex items-center gap-2 px-1 py-2'>
                                          <div className='h-4 w-8 animate-pulse rounded bg-surface-muted' />
                                          <div className='h-7 w-7 animate-pulse rounded-full bg-surface-muted' />
                                          <div className='h-4 w-32 animate-pulse rounded bg-surface-muted' />
                                      </div>
                                  ))
                                : rankingRows.map((row, idx) => (
                                      <button key={`rank-${row.user_id}`} type='button' className='flex w-full items-center gap-2 px-1 py-2 text-left transition hover:bg-surface-muted/70' onClick={() => navigate(`/users/${row.user_id}`)}>
                                          <span className='w-8 text-xs font-semibold text-text-subtle'>#{idx + 1}</span>
                                          <UserAvatar username={row.username} size='sm' />
                                          <div className='min-w-0 flex-1'>
                                              <p className='truncate text-sm text-text'>{row.username}</p>
                                              <p className='truncate text-[11px] text-text-subtle'>{row.affiliation_name?.trim() ? row.affiliation_name : t('common.na')}</p>
                                          </div>
                                          <span className='text-xs font-semibold text-text'>{t('common.pointsShort', { points: row.score })}</span>
                                      </button>
                                  ))}
                            {!loading && rankingRows.length === 0 ? <p className='px-1 py-4 text-sm text-text-subtle'>{t('home.noUserRanking')}</p> : null}
                        </div>
                    </section>

                    <section>
                        <div className='mb-3 flex items-end justify-between gap-2'>
                            <a href='/ranking?tab=affiliations' onClick={(e) => navigate('/ranking?tab=affiliations', e)} className='inline-flex items-center gap-1 text-xl font-semibold text-text transition hover:text-accent md:text-2xl'>
                                <span>{t('home.topAffiliationsTitle')}</span>
                                <LaunchIcon />
                            </a>
                        </div>
                        <div className='relative space-y-1 pb-5'>
                            {loading
                                ? Array.from({ length: 5 }, (_, idx) => (
                                      <div key={`aff-skeleton-${idx}`} className='flex items-center justify-between px-1 py-2'>
                                          <div className='h-4 w-40 animate-pulse rounded bg-surface-muted' />
                                          <div className='h-4 w-20 animate-pulse rounded bg-surface-muted' />
                                      </div>
                                  ))
                                : affiliationRankingRows.map((row, idx) => (
                                      <button
                                          key={`aff-rank-${row.affiliation_id}`}
                                          type='button'
                                          className='flex w-full items-center justify-between gap-3 px-1 py-2 text-left transition hover:bg-surface-muted/70'
                                          onClick={() => navigate('/ranking?tab=affiliations')}
                                      >
                                          <p className='line-clamp-1 text-sm text-text'>
                                              <span className='mr-2 text-xs font-semibold text-text-subtle'>#{idx + 1}</span>
                                              {row.name}
                                          </p>
                                          <p className='shrink-0 text-xs font-semibold text-text'>{t('common.pointsShort', { points: row.score })}</p>
                                      </button>
                                  ))}
                            {!loading && affiliationRankingRows.length === 0 ? <p className='px-1 py-4 text-sm text-text-subtle'>{t('home.noAffiliationRanking')}</p> : null}
                        </div>
                    </section>
                </div>
            </div>
        </section>
    )
}

export default Home
