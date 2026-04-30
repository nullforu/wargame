import { useEffect, useMemo, useState } from 'react'
import { ApiError } from '../lib/api'
import type { Challenge, ChallengeCommentItem, ChallengeSolver, ChallengeVote, LevelVoteCount, PaginationMeta, Stack, Writeup } from '../lib/types'
import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
import { getLocaleTag, useLocale, useTemplate, useT } from '../lib/i18n'
import { navigate } from '../lib/router'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'
import Markdown from '../components/Markdown'
import { LEVEL_VOTE_OPTIONS, normalizeLevel } from '../lib/level'
import VoteModal from './challenge-detail/VoteModal'
import VoteSection from './challenge-detail/VoteSection'
import ChallengeSummaryCard from './challenge-detail/ChallengeSummaryCard'
import ChallengeInfoPanels from './challenge-detail/ChallengeInfoPanels'
import WriteupsSection from './challenge-detail/WriteupsSection'
import SubmitFlagSection from './challenge-detail/SubmitFlagSection'

interface RouteProps {
    routeParams?: Record<string, string>
}

interface SubmissionState {
    status: 'idle' | 'loading' | 'success' | 'error'
    message?: string
}

type VoteModalMode = 'solved' | 'revote'

const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: 5, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const EMPTY_VOTE_PAGINATION: PaginationMeta = { page: 1, page_size: 3, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const EMPTY_WRITEUP_PAGINATION: PaginationMeta = { page: 1, page_size: 5, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const EMPTY_COMMENT_PAGINATION: PaginationMeta = { page: 1, page_size: 5, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
type FirstBloodDurationUnit = 'minute' | 'hour' | 'day' | 'month' | 'year'

const calculateFirstBloodDuration = (createdAt: string, solvedAt: string): { unit: FirstBloodDurationUnit; count: number } | null => {
    const created = new Date(createdAt)
    const solved = new Date(solvedAt)
    if (Number.isNaN(created.getTime()) || Number.isNaN(solved.getTime())) return null

    const diffMs = Math.max(0, solved.getTime() - created.getTime())
    const totalMinutes = Math.max(1, Math.floor(diffMs / (60 * 1000)))
    if (totalMinutes < 60) return { unit: 'minute', count: totalMinutes }

    const totalHours = Math.floor(totalMinutes / 60)
    if (totalHours < 24) return { unit: 'hour', count: totalHours }

    const totalDays = Math.floor(totalHours / 24)
    if (totalDays < 30) return { unit: 'day', count: totalDays }

    const totalMonths = Math.floor(totalDays / 30)
    if (totalMonths < 12) return { unit: 'month', count: totalMonths }

    const totalYears = Math.max(1, Math.floor(totalDays / 365))
    return { unit: 'year', count: totalYears }
}

const ChallengeDetail = ({ routeParams = {} }: RouteProps) => {
    const t = useT()
    const locale = useLocale()
    const pluralRules = useMemo(() => new Intl.PluralRules(locale), [locale])
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const api = useApi()
    const { state: auth } = useAuth()
    const challengeId = useMemo(() => parseRouteId(routeParams.id), [routeParams.id])

    const [challenge, setChallenge] = useState<Challenge | null>(null)
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')

    const [flagInput, setFlagInput] = useState('')
    const [submission, setSubmission] = useState<SubmissionState>({ status: 'idle' })

    const readSolverPageFromQuery = () => {
        if (typeof window === 'undefined') return 1
        const params = new URLSearchParams(window.location.search)
        const value = Number(params.get('solver_page'))
        return Number.isInteger(value) && value > 0 ? value : 1
    }
    const readWriteupPageFromQuery = () => {
        if (typeof window === 'undefined') return 1
        const params = new URLSearchParams(window.location.search)
        const value = Number(params.get('writeup_page'))
        return Number.isInteger(value) && value > 0 ? value : 1
    }
    const [solvers, setSolvers] = useState<ChallengeSolver[]>([])
    const [solverPage, setSolverPage] = useState(readSolverPageFromQuery)
    const [solverPagination, setSolverPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)

    const [downloadLoading, setDownloadLoading] = useState(false)
    const [downloadMessage, setDownloadMessage] = useState('')
    const [stackInfo, setStackInfo] = useState<Stack | null>(null)
    const [stackLoading, setStackLoading] = useState(false)
    const [stackMessage, setStackMessage] = useState('')
    const [votes, setVotes] = useState<ChallengeVote[]>([])
    const [votePage, setVotePage] = useState(1)
    const [votePagination, setVotePagination] = useState<PaginationMeta>(EMPTY_VOTE_PAGINATION)
    const [voteSubmitting, setVoteSubmitting] = useState(false)
    const [voteMessage, setVoteMessage] = useState('')
    const [myVoteLevel, setMyVoteLevel] = useState<number | null>(null)
    const [myVoteLoaded, setMyVoteLoaded] = useState(false)
    const [selectedLevel, setSelectedLevel] = useState<number | null>(null)
    const [isVoteModalOpen, setIsVoteModalOpen] = useState(false)
    const [voteModalMode, setVoteModalMode] = useState<VoteModalMode>('solved')
    const [voteModalLevel, setVoteModalLevel] = useState(1)
    const [writeups, setWriteups] = useState<Writeup[]>([])
    const [writeupPage, setWriteupPage] = useState(readWriteupPageFromQuery)
    const [writeupPagination, setWriteupPagination] = useState<PaginationMeta>(EMPTY_WRITEUP_PAGINATION)
    const [writeupLoading, setWriteupLoading] = useState(false)
    const [writeupError, setWriteupError] = useState('')
    const [canViewWriteupContent, setCanViewWriteupContent] = useState(false)
    const [hasMyWriteup, setHasMyWriteup] = useState(false)
    const [comments, setComments] = useState<ChallengeCommentItem[]>([])
    const [commentPage, setCommentPage] = useState(1)
    const [commentPagination, setCommentPagination] = useState<PaginationMeta>(EMPTY_COMMENT_PAGINATION)
    const [commentLoading, setCommentLoading] = useState(false)
    const [commentError, setCommentError] = useState('')
    const [commentInput, setCommentInput] = useState('')
    const [commentSubmitting, setCommentSubmitting] = useState(false)
    const [editingCommentID, setEditingCommentID] = useState<number | null>(null)
    const [editingCommentContent, setEditingCommentContent] = useState('')

    const pushSolverPageQuery = (nextPage: number) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams(window.location.search)
        if (nextPage > 1) params.set('solver_page', String(nextPage))
        else params.delete('solver_page')
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }
    const pushWriteupPageQuery = (nextPage: number) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams(window.location.search)
        if (nextPage > 1) params.set('writeup_page', String(nextPage))
        else params.delete('writeup_page')
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }

    const loadChallenge = async () => {
        if (!challengeId) return
        setLoading(true)
        setErrorMessage('')

        try {
            const data = await api.challenge(challengeId)
            setChallenge(data)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
            setChallenge(null)
        } finally {
            setLoading(false)
        }
    }

    const loadSolvers = async (page: number) => {
        if (!challengeId) return

        try {
            const data = await api.challengeSolvers(challengeId, page, 5)
            setSolvers(data.solvers)
            setSolverPagination(data.pagination)
        } catch {
            setSolvers([])
            setSolverPagination(EMPTY_PAGINATION)
        }
    }

    const loadVotes = async (page: number) => {
        if (!challengeId) return
        try {
            const data = await api.challengeVotes(challengeId, page, 5)
            setVotes(data.votes)
            setVotePagination(data.pagination)
        } catch {
            setVotes([])
            setVotePagination(EMPTY_VOTE_PAGINATION)
        }
    }

    const loadWriteups = async (page: number) => {
        if (!challengeId) return
        setWriteupLoading(true)
        setWriteupError('')
        try {
            const data = await api.challengeWriteups(challengeId, page, 5)
            setWriteups(data.writeups)
            setWriteupPagination(data.pagination)
            setCanViewWriteupContent(data.can_view_content)
        } catch {
            setWriteups([])
            setWriteupPagination(EMPTY_WRITEUP_PAGINATION)
            setCanViewWriteupContent(false)
            setWriteupError(t('errors.requestFailed'))
        } finally {
            setWriteupLoading(false)
        }
    }

    const loadMyWriteup = async () => {
        if (!challengeId || !auth.user) {
            setHasMyWriteup(false)
            return
        }
        try {
            await api.challengeMyWriteup(challengeId)
            setHasMyWriteup(true)
        } catch {
            setHasMyWriteup(false)
        }
    }

    const loadMyVote = async () => {
        if (!challengeId) return
        try {
            const data = await api.challengeMyVote(challengeId)
            setMyVoteLevel(data.level)
        } catch {
            setMyVoteLevel(null)
        } finally {
            setMyVoteLoaded(true)
        }
    }

    const loadComments = async (page: number) => {
        if (!challengeId) return
        setCommentLoading(true)
        setCommentError('')
        try {
            const data = await api.challengeComments(challengeId, page, 5)
            setComments(data.comments)
            setCommentPagination(data.pagination)
        } catch {
            setComments([])
            setCommentPagination(EMPTY_COMMENT_PAGINATION)
            setCommentError(t('errors.requestFailed'))
        } finally {
            setCommentLoading(false)
        }
    }

    useEffect(() => {
        if (!challengeId) return
        void loadChallenge()
    }, [challengeId])

    const loadStack = async () => {
        if (!challengeId || !challenge || challenge.is_locked || challenge.is_solved || !('stack_enabled' in challenge) || challenge.stack_enabled !== true) return
        setStackLoading(true)
        setStackMessage('')
        try {
            const stack = await api.getStack(challengeId)
            setStackInfo(stack)
        } catch (error) {
            if (error instanceof ApiError && error.status === 404) {
                setStackInfo(null)
            } else {
                setStackMessage(formatApiError(error, t).message)
            }
        } finally {
            setStackLoading(false)
        }
    }

    useEffect(() => {
        if (!challengeId) return
        void loadSolvers(solverPage)
    }, [challengeId, solverPage])

    useEffect(() => {
        if (!challengeId) return
        void loadVotes(votePage)
    }, [challengeId, votePage])

    useEffect(() => {
        if (!challengeId) return
        void loadWriteups(writeupPage)
    }, [challengeId, writeupPage])
    useEffect(() => {
        if (!challengeId) return
        void loadComments(commentPage)
    }, [challengeId, commentPage])

    useEffect(() => {
        if (!challengeId) return
        void loadMyWriteup()
    }, [challengeId, auth.user?.id])

    useEffect(() => {
        if (!challengeId) return
        setMyVoteLoaded(false)
        void loadMyVote()
    }, [challengeId])

    useEffect(() => {
        setVotePage(1)
        setMyVoteLevel(null)
        setMyVoteLoaded(false)
        setSelectedLevel(null)
        setWriteupPage(1)
        setWriteups([])
        setWriteupPagination(EMPTY_WRITEUP_PAGINATION)
        setWriteupLoading(false)
        setWriteupError('')
        setHasMyWriteup(false)
        setCanViewWriteupContent(false)
        setCommentPage(1)
        setComments([])
        setCommentPagination(EMPTY_COMMENT_PAGINATION)
        setCommentError('')
        setCommentInput('')
        setEditingCommentID(null)
        setEditingCommentContent('')
    }, [challengeId])

    useEffect(() => {
        if (!auth.user || !challengeId || !challenge || challenge.is_locked || challenge.is_solved || !('stack_enabled' in challenge) || challenge.stack_enabled !== true) return
        void loadStack()
    }, [auth.user?.id, challengeId, challenge?.id, challenge?.is_locked])

    useEffect(() => {
        const onPopState = () => {
            setSolverPage(readSolverPageFromQuery())
            setWriteupPage(readWriteupPageFromQuery())
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    useEffect(() => {
        if (!myVoteLoaded) return
        if (myVoteLevel !== null) {
            setSelectedLevel(myVoteLevel)
            return
        }
        if (selectedLevel !== null) return
        const currentLevel = normalizeLevel(challenge?.level)
        setSelectedLevel(currentLevel > 0 ? currentLevel : 0)
    }, [challenge?.level, myVoteLevel, myVoteLoaded, selectedLevel])

    const initialVoteLevel = () => {
        if (myVoteLevel !== null && myVoteLevel >= 1 && myVoteLevel <= 10) return myVoteLevel
        const currentLevel = normalizeLevel(challenge?.level)
        return currentLevel >= 1 && currentLevel <= 10 ? currentLevel : 1
    }

    const openVoteModal = (mode: VoteModalMode) => {
        setVoteModalMode(mode)
        setVoteModalLevel(initialVoteLevel())
        setIsVoteModalOpen(true)
    }

    const submitFlag = async () => {
        if (!challengeId || !challenge || challenge.is_locked || challenge.is_active === false) return
        if (!auth.user) return
        if (challenge.is_solved) {
            setSubmission({ status: 'success', message: t('challenge.correct') })
            return
        }
        if (submission.status === 'loading') return

        setSubmission({ status: 'loading' })

        const handleSolveSuccess = async () => {
            setSubmission({ status: 'success', message: t('challenge.correct') })
            setFlagInput('')
            setChallenge((prev) => (prev ? { ...prev, is_solved: true } : prev))
            setCanViewWriteupContent(true)
            await Promise.all([loadChallenge(), loadSolvers(solverPage), loadWriteups(writeupPage)])
            openVoteModal('solved')
        }

        try {
            const result = await api.submitFlag(challengeId, flagInput)
            if (result.correct) {
                await handleSolveSuccess()
            } else {
                setSubmission({ status: 'error', message: t('challenge.incorrect') })
            }
        } catch (error) {
            if (error instanceof ApiError && error.status === 409) {
                await handleSolveSuccess()
                return
            }
            setSubmission({ status: 'error', message: formatApiError(error, t).message })
        }
    }

    const submitLevelVote = async (level: number, closeModalOnSuccess = false) => {
        if (!challengeId || voteSubmitting || !auth.user) return
        setVoteSubmitting(true)
        setVoteMessage('')
        try {
            await api.voteChallengeLevel(challengeId, level)
            setMyVoteLevel(level)
            setMyVoteLoaded(true)
            setSelectedLevel(level)
            setVoteModalLevel(level)
            setVoteMessage(t('challenge.voteSubmitted'))
            await loadChallenge()
            await loadVotes(votePage)
            if (closeModalOnSuccess) {
                setIsVoteModalOpen(false)
            }
        } catch (error) {
            setVoteMessage(formatApiError(error, t).message)
        } finally {
            setVoteSubmitting(false)
        }
    }

    const downloadFile = async () => {
        if (!challengeId || !challenge || !('has_file' in challenge) || !challenge.has_file || downloadLoading || !auth.user) return
        setDownloadLoading(true)
        setDownloadMessage('')
        try {
            const result = await api.requestChallengeFileDownload(challengeId)
            window.open(result.url, '_blank', 'noopener')
        } catch (error) {
            setDownloadMessage(formatApiError(error, t).message)
        } finally {
            setDownloadLoading(false)
        }
    }

    const createStack = async () => {
        if (!challengeId || !challenge || challenge.is_locked || !('stack_enabled' in challenge) || challenge.stack_enabled !== true || stackLoading || !auth.user) return
        setStackLoading(true)
        setStackMessage('')
        try {
            const created = await api.createStack(challengeId)
            setStackInfo(created)
        } catch (error) {
            setStackMessage(formatApiError(error, t).message)
        } finally {
            setStackLoading(false)
        }
    }

    const deleteStack = async () => {
        if (!challengeId || !challenge || challenge.is_locked || !('stack_enabled' in challenge) || challenge.stack_enabled !== true || stackLoading || !auth.user) return
        setStackLoading(true)
        setStackMessage('')
        try {
            await api.deleteStack(challengeId)
            setStackInfo(null)
        } catch (error) {
            setStackMessage(formatApiError(error, t).message)
        } finally {
            setStackLoading(false)
        }
    }

    const formatTimestamp = (value: string) => formatDateTime(value, localeTag)
    const formatWriteupDate = (value: string) => {
        const date = new Date(value)
        if (Number.isNaN(date.getTime())) return t('common.na')
        const yyyy = date.getFullYear()
        const mm = String(date.getMonth() + 1).padStart(2, '0')
        const dd = String(date.getDate()).padStart(2, '0')
        return `${yyyy}.${mm}.${dd}`
    }
    const formatCompactDateTime = (value: string) => {
        const date = new Date(value)
        if (Number.isNaN(date.getTime())) return t('common.na')
        const yyyy = date.getFullYear()
        const mm = String(date.getMonth() + 1).padStart(2, '0')
        const dd = String(date.getDate()).padStart(2, '0')
        const hh = String(date.getHours()).padStart(2, '0')
        const min = String(date.getMinutes()).padStart(2, '0')
        return `${yyyy}-${mm}-${dd} ${hh}:${min}`
    }
    const firstBloodSolver = useMemo(() => {
        if (challenge && 'first_blood' in challenge && challenge.first_blood) return challenge.first_blood
        return solvers.find((solver) => solver.is_first_blood) ?? null
    }, [challenge, solvers])
    const firstBloodDuration = useMemo(() => {
        if (!challenge?.created_at || !firstBloodSolver?.solved_at) return null
        return calculateFirstBloodDuration(challenge.created_at, firstBloodSolver.solved_at)
    }, [challenge?.created_at, firstBloodSolver?.solved_at])
    const firstBloodDurationKey = useMemo(() => {
        if (!firstBloodDuration) return ''
        const pluralCategory = pluralRules.select(firstBloodDuration.count)
        const suffix = pluralCategory === 'one' ? 'one' : 'other'
        return `challenge.firstBloodSolvedAfter.${firstBloodDuration.unit}.${suffix}`
    }, [firstBloodDuration, pluralRules])
    const renderFirstBloodDuration = useTemplate(firstBloodDurationKey || 'challenge.firstBloodSolvedAfter.minute.other')
    const firstBloodSolvedAfterLabel = firstBloodDuration ? renderFirstBloodDuration({ count: firstBloodDuration.count }) : null
    const currentLevel = normalizeLevel(challenge?.level)
    const levelLabel = currentLevel > 0 ? String(currentLevel) : t('level.unknown')
    const voteCountsByLevel = useMemo(() => {
        const source = challenge && 'level_vote_counts' in challenge ? (challenge.level_vote_counts ?? []) : []
        const mapped = new Map<number, number>()
        source.forEach((item: LevelVoteCount) => {
            mapped.set(item.level, item.count)
        })
        return mapped
    }, [challenge])
    const maxVoteCount = useMemo(() => {
        let max = 0
        LEVEL_VOTE_OPTIONS.forEach((level) => {
            max = Math.max(max, voteCountsByLevel.get(level) ?? 0)
        })
        return max
    }, [voteCountsByLevel])
    const voteLevelDescription = useMemo(() => {
        if (voteModalLevel <= 2) return t('challenge.voteLevelDescription.1_2')
        if (voteModalLevel <= 4) return t('challenge.voteLevelDescription.3_4')
        if (voteModalLevel <= 6) return t('challenge.voteLevelDescription.5_6')
        if (voteModalLevel <= 8) return t('challenge.voteLevelDescription.7_8')
        return t('challenge.voteLevelDescription.9_10')
    }, [voteModalLevel, t])
    const voteModalTitle = voteModalMode === 'solved' ? t('challenge.voteModalSolvedTitle') : t('challenge.voteModalRevoteTitle')
    const submitComment = async () => {
        if (!challengeId || !auth.user || commentSubmitting) return
        setCommentSubmitting(true)
        try {
            await api.createChallengeComment(challengeId, commentInput)
            setCommentInput('')
            setCommentPage(1)
            await loadComments(1)
        } catch (error) {
            setCommentError(formatApiError(error, t).message)
        } finally {
            setCommentSubmitting(false)
        }
    }
    const updateComment = async (commentID: number) => {
        if (!auth.user || commentSubmitting) return
        setCommentSubmitting(true)
        try {
            await api.updateChallengeComment(commentID, { content: editingCommentContent })
            setEditingCommentID(null)
            setEditingCommentContent('')
            await loadComments(commentPage)
        } catch (error) {
            setCommentError(formatApiError(error, t).message)
        } finally {
            setCommentSubmitting(false)
        }
    }
    const deleteComment = async (commentID: number) => {
        if (!auth.user || commentSubmitting) return
        setCommentSubmitting(true)
        try {
            await api.deleteChallengeComment(commentID)
            await loadComments(commentPage)
        } catch (error) {
            setCommentError(formatApiError(error, t).message)
        } finally {
            setCommentSubmitting(false)
        }
    }
    if (!challengeId) {
        return (
            <section className='animate'>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{t('errors.invalid')}</div>
            </section>
        )
    }

    if (loading) {
        return (
            <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
                <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                    <div className='min-w-0 space-y-4'>
                        <div className='rounded-2xl sm:p-5'>
                            <div className='animate-pulse space-y-4'>
                                <div className='h-5 w-40 rounded bg-surface-muted' />
                                <div className='space-y-2'>
                                    <div className='h-4 w-full rounded bg-surface-muted' />
                                    <div className='h-4 w-11/12 rounded bg-surface-muted' />
                                    <div className='h-4 w-4/5 rounded bg-surface-muted' />
                                </div>
                                <div className='mt-8 h-10 w-full rounded bg-surface-muted' />
                            </div>
                        </div>
                    </div>

                    <div className='hidden lg:block'>
                        <div className='space-y-8'>
                            <div className='rounded-2xl border border-border/20 bg-surface p-5 shadow-sm animate-pulse space-y-3'>
                                <div className='h-6 w-16 rounded bg-surface-muted' />
                                <div className='h-9 w-full rounded bg-surface-muted' />
                                <div className='h-5 w-1/3 rounded bg-surface-muted' />
                                <div className='h-5 w-4/5 rounded bg-surface-muted' />
                            </div>

                            <section className='space-y-3 px-1'>
                                <div className='h-7 w-24 rounded bg-surface-muted animate-pulse' />
                                <div className='rounded-2xl bg-surface/70 p-2 animate-pulse'>
                                    <div className='flex items-start gap-3.75'>
                                        <div className='h-10 w-10 rounded-full bg-surface-muted' />
                                        <div className='min-w-0 flex-1 space-y-2'>
                                            <div className='h-5 w-1/2 rounded bg-surface-muted' />
                                            <div className='h-4 w-1/3 rounded bg-surface-muted' />
                                            <div className='h-4 w-2/3 rounded bg-surface-muted' />
                                        </div>
                                    </div>
                                </div>
                            </section>
                        </div>
                    </div>
                </div>
            </section>
        )
    }

    if (errorMessage || !challenge) {
        return (
            <section className='animate space-y-3'>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{errorMessage || t('errors.notFound')}</div>
            </section>
        )
    }

    const detail = challenge.is_locked ? null : challenge
    const isChallengeActive = challenge.is_locked ? false : challenge.is_active !== false
    const isSubmissionDisabled = !auth.user || !isChallengeActive || challenge.is_solved || submission.status === 'loading'
    const createdSummary = challenge.created_at ? formatCompactDateTime(challenge.created_at) : t('common.na')
    const voteSubmittedMessage = t('challenge.voteSubmitted')
    return (
        <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
            <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                <div className='min-w-0 space-y-4'>
                    <div className='lg:hidden'>
                        <ChallengeSummaryCard challenge={challenge} levelLabel={levelLabel} createdSummary={createdSummary} t={t} />
                    </div>

                    <div className='min-w-0 rounded-2xl sm:p-5'>
                        <h2 className='text-base font-semibold text-text'>{t('common.description')}</h2>

                        {challenge.is_locked ? (
                            <div className='mt-3 rounded-xl bg-warning/10 p-4 text-sm text-warning'>
                                <p>{t('challenge.lockedNotice')}</p>

                                <button
                                    className='mt-3 rounded-md bg-warning px-3 py-1.5 text-xs text-white hover:bg-warning-strong disabled:opacity-60'
                                    onClick={() => {
                                        if (challenge.previous_challenge_id) {
                                            navigate(`/challenges/${challenge.previous_challenge_id}`)
                                        }
                                    }}
                                    disabled={!challenge.previous_challenge_id}
                                >
                                    {challenge.previous_challenge_category} - {challenge.previous_challenge_title}
                                </button>
                            </div>
                        ) : (
                            <div className='mt-3 min-w-0'>
                                <Markdown className='text-sm text-text' content={detail?.description ?? ''} />
                            </div>
                        )}

                        <div className='mt-12 space-y-8 lg:hidden'>
                            <ChallengeInfoPanels
                                challenge={challenge}
                                firstBloodSolver={firstBloodSolver}
                                firstBloodSolvedAfterLabel={firstBloodSolvedAfterLabel}
                                solvers={solvers}
                                solverPagination={solverPagination}
                                solverPage={solverPage}
                                formatTimestamp={formatTimestamp}
                                onSetSolverPage={setSolverPage}
                                onPushSolverPageQuery={pushSolverPageQuery}
                                comments={comments}
                                commentPagination={commentPagination}
                                commentPage={commentPage}
                                commentLoading={commentLoading}
                                commentError={commentError}
                                commentInput={commentInput}
                                commentSubmitting={commentSubmitting}
                                editingCommentID={editingCommentID}
                                editingCommentContent={editingCommentContent}
                                isAuthenticated={Boolean(auth.user)}
                                currentUserID={auth.user?.id}
                                onCommentInputChange={setCommentInput}
                                onCreateComment={() => void submitComment()}
                                onEditStart={(id, content) => {
                                    setEditingCommentID(id)
                                    setEditingCommentContent(content)
                                }}
                                onEditCancel={() => setEditingCommentID(null)}
                                onEditContentChange={setEditingCommentContent}
                                onUpdateComment={(id) => void updateComment(id)}
                                onDeleteComment={(id) => void deleteComment(id)}
                                onSetCommentPage={setCommentPage}
                                t={t}
                            />
                        </div>

                        {!challenge.is_locked && detail?.has_file && (
                            <div className='mt-12'>
                                <button
                                    onClick={downloadFile}
                                    disabled={downloadLoading || !auth.user}
                                    className='w-full h-10 flex items-center justify-center gap-2 rounded-sm border border-border bg-surface-muted px-4 py-4 text-sm font-medium text-accent hover:bg-surface-subtle transition disabled:opacity-60'
                                >
                                    <svg xmlns='http://www.w3.org/2000/svg' className='h-4 w-4' fill='none' viewBox='0 0 24 24' stroke='currentColor' strokeWidth={2}>
                                        <path strokeLinecap='round' strokeLinejoin='round' d='M12 3v12m0 0l-4-4m4 4l4-4M4 17h16' />
                                    </svg>

                                    {downloadLoading ? t('challenge.downloadPreparing') : t('challenge.download')}
                                </button>
                                {!auth.user ? (
                                    <p className='mt-2 text-xs text-warning'>
                                        {t('challenge.fileLoginRequired')}{' '}
                                        <a className='underline cursor-pointer' href='/login' onClick={(e) => navigate('/login', e)}>
                                            {t('auth.loginLink')}
                                        </a>
                                    </p>
                                ) : null}

                                {downloadMessage && <p className='mt-2 text-xs text-danger'>{downloadMessage}</p>}
                            </div>
                        )}

                        {!challenge.is_locked && !challenge.is_solved && 'stack_enabled' in challenge && challenge.stack_enabled ? (
                            <div className='rounded-md border border-border/30 bg-surface p-4 sm:p-5 shadow-sm mt-8'>
                                <div className='flex items-center justify-between gap-2'>
                                    <h2 className='text-base font-semibold text-text'>{t('challenge.stackInstance')}</h2>
                                    {auth.user ? (
                                        <button className='rounded-lg bg-surface-muted px-3 py-1.5 text-xs text-text hover:bg-surface-subtle disabled:opacity-60' onClick={() => void loadStack()} disabled={stackLoading}>
                                            {t('common.refresh')}
                                        </button>
                                    ) : null}
                                </div>

                                {auth.user ? (
                                    stackInfo ? (
                                        <div className='mt-3 space-y-1.5 text-sm text-text-muted'>
                                            <p>
                                                {t('challenge.stackStatus')} <span className='text-text'>{stackInfo.status}</span>
                                            </p>
                                            <p>
                                                {t('challenge.stackEndpoint')}{' '}
                                                <span className='break-all text-text'>
                                                    {stackInfo.node_public_ip && stackInfo.ports.length > 0 ? stackInfo.ports.map((port) => `${port.protocol} ${stackInfo.node_public_ip}:${port.node_port}`).join(', ') : t('common.pending')}
                                                </span>
                                            </p>
                                            <p>
                                                {t('challenge.stackPorts')}{' '}
                                                <span className='text-text'>{stackInfo.ports.length > 0 ? stackInfo.ports.map((port) => `${port.container_port}/${port.protocol}`).join(', ') : t('common.pending')}</span>
                                            </p>
                                            <p>
                                                {t('challenge.stackTtl')} <span className='text-text'>{stackInfo.ttl_expires_at ? formatTimestamp(stackInfo.ttl_expires_at) : t('common.pending')}</span>
                                            </p>
                                        </div>
                                    ) : (
                                        <p className='mt-3 text-sm text-text-muted'>{t('challenge.stackNoActive')}</p>
                                    )
                                ) : (
                                    <p className='mt-3 text-sm text-warning'>
                                        {t('challenge.stackLoginRequired')}{' '}
                                        <a className='underline cursor-pointer' href='/login' onClick={(e) => navigate('/login', e)}>
                                            {t('auth.loginLink')}
                                        </a>
                                    </p>
                                )}

                                {auth.user ? (
                                    <div className='mt-4 flex flex-wrap gap-2'>
                                        {stackInfo ? (
                                            <button className='rounded-lg border border-danger/20 px-3 py-2 text-sm text-danger hover:border-danger/40 disabled:opacity-60' onClick={deleteStack} disabled={stackLoading}>
                                                {stackLoading ? t('challenge.stackWorking') : t('challenge.deleteStack')}
                                            </button>
                                        ) : (
                                            <button className='rounded-lg bg-accent px-3 py-2 text-sm text-white hover:bg-accent-strong disabled:opacity-60' onClick={createStack} disabled={stackLoading || challenge.is_solved}>
                                                {stackLoading ? t('challenge.stackWorking') : t('challenge.createStack')}
                                            </button>
                                        )}
                                    </div>
                                ) : null}

                                {stackMessage ? <p className='mt-2 text-xs text-danger'>{stackMessage}</p> : null}
                            </div>
                        ) : null}

                        {!challenge.is_locked && !challenge.is_solved ? (
                            <SubmitFlagSection
                                flagInput={flagInput}
                                isSubmissionDisabled={isSubmissionDisabled}
                                submission={submission}
                                isAuthenticated={Boolean(auth.user)}
                                onFlagInputChange={setFlagInput}
                                onSubmit={() => void submitFlag()}
                                t={t}
                            />
                        ) : null}

                        <VoteSection
                            challenge={challenge}
                            isAuthenticated={Boolean(auth.user)}
                            votePagination={votePagination}
                            voteCountsByLevel={voteCountsByLevel}
                            maxVoteCount={maxVoteCount}
                            selectedLevel={selectedLevel}
                            votes={votes}
                            voteMessage={voteMessage}
                            voteSubmittedMessage={voteSubmittedMessage}
                            formatTimestamp={formatTimestamp}
                            onOpenRevote={() => openVoteModal('revote')}
                            onPrevVotePage={() => setVotePage((prev) => Math.max(1, prev - 1))}
                            onNextVotePage={() => setVotePage((prev) => prev + 1)}
                            t={t}
                        />

                        {!challenge.is_locked ? (
                            <WriteupsSection
                                challengeId={challengeId}
                                challengeSolved={challenge.is_solved}
                                hasMyWriteup={hasMyWriteup}
                                writeups={writeups}
                                writeupLoading={writeupLoading}
                                writeupError={writeupError}
                                canViewWriteupContent={canViewWriteupContent}
                                writeupPage={writeupPage}
                                writeupPagination={writeupPagination}
                                formatWriteupDate={formatWriteupDate}
                                onSetWriteupPage={setWriteupPage}
                                onPushWriteupPageQuery={pushWriteupPageQuery}
                                t={t}
                            />
                        ) : null}
                    </div>
                </div>

                <aside className='hidden lg:block lg:sticky'>
                    <div className='space-y-8'>
                        <ChallengeSummaryCard challenge={challenge} levelLabel={levelLabel} createdSummary={createdSummary} t={t} />
                        <ChallengeInfoPanels
                            challenge={challenge}
                            firstBloodSolver={firstBloodSolver}
                            firstBloodSolvedAfterLabel={firstBloodSolvedAfterLabel}
                            solvers={solvers}
                            solverPagination={solverPagination}
                            solverPage={solverPage}
                            formatTimestamp={formatTimestamp}
                            onSetSolverPage={setSolverPage}
                            onPushSolverPageQuery={pushSolverPageQuery}
                            comments={comments}
                            commentPagination={commentPagination}
                            commentPage={commentPage}
                            commentLoading={commentLoading}
                            commentError={commentError}
                            commentInput={commentInput}
                            commentSubmitting={commentSubmitting}
                            editingCommentID={editingCommentID}
                            editingCommentContent={editingCommentContent}
                            isAuthenticated={Boolean(auth.user)}
                            currentUserID={auth.user?.id}
                            onCommentInputChange={setCommentInput}
                            onCreateComment={() => void submitComment()}
                            onEditStart={(id, content) => {
                                setEditingCommentID(id)
                                setEditingCommentContent(content)
                            }}
                            onEditCancel={() => setEditingCommentID(null)}
                            onEditContentChange={setEditingCommentContent}
                            onUpdateComment={(id) => void updateComment(id)}
                            onDeleteComment={(id) => void deleteComment(id)}
                            onSetCommentPage={setCommentPage}
                            t={t}
                        />
                    </div>
                </aside>
            </div>
            <VoteModal
                isOpen={isVoteModalOpen}
                mode={voteModalMode}
                title={voteModalTitle}
                subtitle={t('challenge.voteModalSolvedSubtitle')}
                hint={t('challenge.voteModalHint')}
                headline={`${challenge.title}`}
                level={voteModalLevel}
                description={voteLevelDescription}
                submitting={voteSubmitting}
                canSubmit={Boolean(auth.user && challenge.is_solved)}
                cancelLabel={t('common.cancel')}
                submitLabel={voteSubmitting ? t('challenge.submitting') : t('challenge.voteSubmitButton')}
                voteAriaLabel={t('challenge.voteTitle')}
                onClose={() => setIsVoteModalOpen(false)}
                onLevelChange={(next) => setVoteModalLevel(Math.max(1, Math.min(10, next)))}
                onSubmit={() => void submitLevelVote(voteModalLevel, true)}
            />
        </section>
    )
}

export default ChallengeDetail
