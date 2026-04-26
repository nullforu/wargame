import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Affiliation, PaginationMeta, Stack, UserDetail, SolvedChallenge } from '../lib/types'
import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
import { navigate } from '../lib/router'
import ProfileHeader from '../components/UserProfile/ProfileHeader'
import AccountCard from '../components/UserProfile/AccountCard'
import ActiveStacksCard from '../components/UserProfile/ActiveStacksCard'
import StatisticsCard from '../components/UserProfile/StatisticsCard'
import { getLocaleTag, useLocale, useT } from '../lib/i18n'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'
import LoginRequired from '../components/LoginRequired'

interface RouteProps {
    routeParams?: Record<string, string>
}

const UserProfile = ({ routeParams = {} }: RouteProps) => {
    const t = useT()
    const api = useApi()
    const { state: auth } = useAuth()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const [user, setUser] = useState<UserDetail | null>(null)
    const [solved, setSolved] = useState<SolvedChallenge[]>([])
    const readSolvedPageFromQuery = () => {
        if (typeof window === 'undefined') return 1
        const params = new URLSearchParams(window.location.search)
        const value = Number(params.get('solved_page'))
        return Number.isInteger(value) && value > 0 ? value : 1
    }
    const [solvedPage, setSolvedPage] = useState(readSolvedPageFromQuery)
    const [solvedPagination, setSolvedPagination] = useState({ page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [stacks, setStacks] = useState<Stack[]>([])
    const [stacksLoading, setStacksLoading] = useState(false)
    const [stacksError, setStacksError] = useState('')
    const [stackDeletingId, setStackDeletingId] = useState<number | null>(null)
    const [editingUsername, setEditingUsername] = useState(false)
    const [usernameInput, setUsernameInput] = useState('')
    const [savingUsername, setSavingUsername] = useState(false)
    const [editingBio, setEditingBio] = useState(false)
    const [bioInput, setBioInput] = useState('')
    const [savingBio, setSavingBio] = useState(false)
    const [editingAffiliation, setEditingAffiliation] = useState(false)
    const [savingAffiliation, setSavingAffiliation] = useState(false)
    const [affiliationPage, setAffiliationPage] = useState(1)
    const [affiliationQuery, setAffiliationQuery] = useState('')
    const [debouncedAffiliationQuery, setDebouncedAffiliationQuery] = useState('')
    const [affiliations, setAffiliations] = useState<Affiliation[]>([])
    const [affiliationPagination, setAffiliationPagination] = useState<PaginationMeta>({ page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
    const [selectedAffiliationID, setSelectedAffiliationID] = useState<number | null>(null)
    const [loadingAffiliations, setLoadingAffiliations] = useState(false)
    const lastStacksLoadedForUserIdRef = useRef<number | null>(null)

    const routeUserId = useMemo(() => parseRouteId(routeParams.id), [routeParams.id])
    const isOwnProfile = useMemo(() => (auth.user ? !routeUserId || routeUserId === auth.user.id : false), [auth.user, routeUserId])
    const showBackButton = !!routeParams.id
    const activeStacks = useMemo(() => stacks.filter((stack) => !['stopped', 'failed', 'node_deleted'].includes(stack.status)), [stacks])
    const targetUserId = routeUserId ?? auth.user?.id ?? null
    const totalSolvedPoints = useMemo(() => solved.reduce((sum, item) => sum + item.points, 0), [solved])
    const canViewProfile = routeUserId !== null || auth.user !== null

    const formatOptionalDateTime = useCallback((value?: string | null) => (value ? formatDateTime(value, localeTag) : t('common.na')), [localeTag, t])

    const formatSolvedDateTime = useCallback((value: string) => formatDateTime(value, localeTag), [localeTag])

    const loadUserProfile = useCallback(
        async (userId: number) => {
            setLoading(true)
            setErrorMessage('')
            setSolved([])
            setSolvedPagination({ page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false })

            try {
                const [userDetail, solvedData] = await Promise.all([api.user(userId), api.userSolved(userId, solvedPage, 20)])
                setUser(userDetail)
                setSolved(solvedData.solved)
                setSolvedPagination(solvedData.pagination)
            } catch (error) {
                setErrorMessage(formatApiError(error, t).message)
            } finally {
                setLoading(false)
            }
        },
        [api, solvedPage, t],
    )

    const loadStacks = useCallback(async () => {
        if (!isOwnProfile) return

        setStacksLoading(true)
        setStacksError('')

        try {
            const response = await api.stacks()
            setStacks(response.stacks)
        } catch (error) {
            setStacksError(formatApiError(error, t).message)
        } finally {
            setStacksLoading(false)
        }
    }, [api, isOwnProfile, t])

    const deleteStack = useCallback(
        async (challengeId: number) => {
            if (stackDeletingId !== null) return

            setStackDeletingId(challengeId)
            setStacksError('')

            try {
                await api.deleteStack(challengeId)
                await loadStacks()
            } catch (error) {
                setStacksError(formatApiError(error, t).message)
            } finally {
                setStackDeletingId(null)
            }
        },
        [api, loadStacks, stackDeletingId, t],
    )

    const saveUsername = useCallback(async () => {
        if (!user) return

        setSavingUsername(true)
        setErrorMessage('')

        try {
            const updated = await api.updateMe({ username: usernameInput.trim() })
            setUser(updated)
            setEditingUsername(false)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSavingUsername(false)
        }
    }, [api, t, user, usernameInput])

    const loadAffiliations = useCallback(async () => {
        setLoadingAffiliations(true)
        setErrorMessage('')
        try {
            const data = debouncedAffiliationQuery ? await api.searchAffiliations(debouncedAffiliationQuery, affiliationPage, 10) : await api.affiliations(affiliationPage, 10)
            setAffiliations(data.affiliations)
            setAffiliationPagination(data.pagination)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
            setAffiliations([])
            setAffiliationPagination({ page: 1, page_size: 10, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
        } finally {
            setLoadingAffiliations(false)
        }
    }, [affiliationPage, api, debouncedAffiliationQuery, t])

    const saveAffiliation = useCallback(async () => {
        if (!user) return
        setSavingAffiliation(true)
        setErrorMessage('')
        try {
            const updated = await api.updateMe({ affiliation_id: selectedAffiliationID })
            setUser(updated)
            setEditingAffiliation(false)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSavingAffiliation(false)
        }
    }, [api, selectedAffiliationID, t, user])

    const saveBio = useCallback(async () => {
        if (!user) return
        setSavingBio(true)
        setErrorMessage('')
        try {
            const trimmed = bioInput.trim()
            const updated = await api.updateMe({ bio: trimmed === '' ? null : trimmed })
            setUser(updated)
            setEditingBio(false)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSavingBio(false)
        }
    }, [api, bioInput, t, user])

    const pushSolvedPageQuery = useCallback((nextPage: number) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams(window.location.search)
        if (nextPage > 1) params.set('solved_page', String(nextPage))
        else params.delete('solved_page')
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }, [])

    const backToUsersURL = useMemo(() => {
        if (typeof window === 'undefined') return '/users'
        const params = new URLSearchParams(window.location.search)
        params.delete('solved_page')
        const query = params.toString()
        return query ? `/users?${query}` : '/users'
    }, [routeParams.id, solvedPage])

    useEffect(() => {
        if (user && isOwnProfile) {
            setUsernameInput(user.username)
            setBioInput(user.bio ?? '')
            setSelectedAffiliationID(user.affiliation_id)
            setAffiliationQuery('')
            setDebouncedAffiliationQuery('')
            setAffiliationPage(1)
        }
    }, [user, isOwnProfile])

    useEffect(() => {
        if (!editingAffiliation || !isOwnProfile) return
        loadAffiliations()
    }, [editingAffiliation, isOwnProfile, loadAffiliations])

    useEffect(() => {
        if (!editingAffiliation || !isOwnProfile) return
        const timer = window.setTimeout(() => {
            setDebouncedAffiliationQuery(affiliationQuery.trim())
        }, 250)
        return () => window.clearTimeout(timer)
    }, [affiliationQuery, editingAffiliation, isOwnProfile])

    useEffect(() => {
        if (targetUserId === null) return
        void loadUserProfile(targetUserId)
    }, [loadUserProfile, targetUserId, solvedPage])

    useEffect(() => {
        const onPopState = () => {
            setSolvedPage(readSolvedPageFromQuery())
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    useEffect(() => {
        if (!isOwnProfile) {
            lastStacksLoadedForUserIdRef.current = null
            return
        }

        if (!auth.user) return
        if (lastStacksLoadedForUserIdRef.current === auth.user.id) return

        lastStacksLoadedForUserIdRef.current = auth.user.id
        loadStacks()
    }, [auth.user, isOwnProfile, loadStacks])

    return (
        <section className='animate'>
            {showBackButton ? (
                <div className='mb-4 md:mb-6'>
                    <button className='inline-flex items-center gap-2 text-sm text-text-muted hover:text-accent cursor-pointer' onClick={() => navigate(backToUsersURL)}>
                        <svg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                            <path d='m15 18-6-6 6-6' />
                        </svg>
                        {t('profile.backToUsers')}
                    </button>
                </div>
            ) : null}

            {!canViewProfile ? (
                <LoginRequired title={t('profile.title')} />
            ) : loading ? (
                <div className='rounded-none border-0 bg-transparent p-3 shadow-none md:rounded-2xl md:border md:border-border md:bg-surface md:p-8'>
                    <p className='text-center text-sm text-text-muted'>{t('common.loading')}</p>
                </div>
            ) : errorMessage ? (
                <div className='rounded-none border-0 bg-danger/10 p-3 shadow-none md:rounded-2xl md:border md:border-danger/30 md:p-8'>
                    <p className='text-center text-sm text-danger'>{errorMessage}</p>
                </div>
            ) : user ? (
                <div>
                    <ProfileHeader user={user} />

                    {isOwnProfile ? (
                        <>
                            <AccountCard
                                user={user}
                                authEmail={auth.user?.email}
                                savingUsername={savingUsername}
                                onSave={saveUsername}
                                editingUsername={editingUsername}
                                usernameInput={usernameInput}
                                onEditingUsernameChange={setEditingUsername}
                                onUsernameInputChange={setUsernameInput}
                                editingBio={editingBio}
                                bioInput={bioInput}
                                savingBio={savingBio}
                                onEditingBioChange={setEditingBio}
                                onBioInputChange={setBioInput}
                                onSaveBio={saveBio}
                                editingAffiliation={editingAffiliation}
                                onEditingAffiliationChange={setEditingAffiliation}
                                affiliationQuery={affiliationQuery}
                                onAffiliationQueryChange={(value) => {
                                    setAffiliationQuery(value)
                                    setAffiliationPage(1)
                                }}
                                selectedAffiliationID={selectedAffiliationID}
                                onSelectedAffiliationIDChange={setSelectedAffiliationID}
                                affiliations={affiliations}
                                affiliationPagination={affiliationPagination}
                                loadingAffiliations={loadingAffiliations}
                                savingAffiliation={savingAffiliation}
                                onAffiliationPageChange={setAffiliationPage}
                                onSaveAffiliation={saveAffiliation}
                            />

                            <ActiveStacksCard
                                activeStacks={activeStacks}
                                stacksError={stacksError}
                                stacksLoading={stacksLoading}
                                stackDeletingId={stackDeletingId}
                                onRefresh={loadStacks}
                                onDelete={deleteStack}
                                formatOptionalDateTime={formatOptionalDateTime}
                            />
                        </>
                    ) : null}

                    <div className='mt-8 rounded-none border-0 bg-transparent p-0 shadow-none md:rounded-lg md:border md:border-border md:bg-surface md:p-6'>
                        <div className='flex flex-wrap items-center justify-between gap-2'>
                            <h3 className='text-lg text-text'>{t('profile.solvedChallenges')}</h3>
                            <span className='text-sm text-text-muted'>{solved.length === 1 ? t('profile.problemSingular', { count: solved.length }) : t('profile.problemPlural', { count: solved.length })}</span>
                        </div>

                        <div className='mt-6 space-y-3'>
                            {solved.map((item) => (
                                <div key={item.challenge_id} className='rounded-none border-0 bg-transparent p-3 md:rounded-xl md:border md:border-border md:bg-surface-muted md:p-5'>
                                    <h4 className='text-base font-medium text-text'>
                                        {item.title}
                                        <span className='ml-2 text-xs text-accent'>{t('common.pointsShort', { points: item.points })}</span>
                                    </h4>
                                    <p className='mt-2 text-sm text-text-muted'>{t('profile.solvedAt', { time: formatSolvedDateTime(item.solved_at) })}</p>
                                </div>
                            ))}

                            {solved.length === 0 ? (
                                <div className='rounded-none border-0 bg-surface-muted p-5 text-center md:rounded-xl md:border md:border-border md:p-8'>
                                    <p className='text-sm text-text-muted'>{t('profile.noSolved')}</p>
                                </div>
                            ) : null}
                        </div>
                        {solvedPagination.total_pages > 0 ? (
                            <div className='mt-3 flex flex-wrap items-center justify-end gap-2 text-xs text-text-muted md:px-3'>
                                <button
                                    className='rounded-md border border-border px-2 py-1 disabled:opacity-50'
                                    disabled={!solvedPagination.has_prev}
                                    onClick={() => {
                                        const nextPage = Math.max(1, solvedPage - 1)
                                        setSolvedPage(nextPage)
                                        pushSolvedPageQuery(nextPage)
                                    }}
                                >
                                    {t('common.previous')}
                                </button>
                                <span>
                                    {solvedPagination.page} / {solvedPagination.total_pages || 1}
                                </span>
                                <button
                                    className='rounded-md border border-border px-2 py-1 disabled:opacity-50'
                                    disabled={!solvedPagination.has_next}
                                    onClick={() => {
                                        const nextPage = solvedPage + 1
                                        setSolvedPage(nextPage)
                                        pushSolvedPageQuery(nextPage)
                                    }}
                                >
                                    {t('common.next')}
                                </button>
                            </div>
                        ) : null}
                    </div>

                    {solved.length > 0 ? <StatisticsCard totalPoints={totalSolvedPoints} solvedCount={solvedPagination.total_count || solved.length} /> : null}
                </div>
            ) : null}
        </section>
    )
}

export default UserProfile
