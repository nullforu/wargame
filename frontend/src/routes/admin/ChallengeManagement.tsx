import { Fragment, useEffect, useRef, useState } from 'react'
import { uploadPresignedPost } from '../../lib/api'
import { CHALLENGE_CATEGORIES } from '../../lib/constants'
import { formatApiError, isZipFile, type FieldErrors } from '../../lib/utils'
import type { Challenge, ChallengeDetail, ChallengeUpdatePayload, TargetPortSpec } from '../../lib/types'

type TargetPortRow = TargetPortSpec & { id: string }
import FormMessage from '../../components/FormMessage'
import { getCategoryKey, useT } from '../../lib/i18n'
import { useApi } from '../../lib/useApi'
import MonacoEditor from '../../components/MonacoEditor'

type ActiveFilter = 'all' | 'active' | 'inactive'
type SortFilter = 'latest' | 'oldest' | 'most_solved' | 'least_solved'

const ChallengeManagement = () => {
    const t = useT()
    const api = useApi()
    const [challenges, setChallenges] = useState<Challenge[]>([])
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')
    const [expandedChallengeId, setExpandedChallengeId] = useState<number | null>(null)
    const [manageLoading, setManageLoading] = useState(false)
    const [manageFieldErrors, setManageFieldErrors] = useState<FieldErrors>({})
    const [editingField, setEditingField] = useState<'title' | 'description' | 'category' | 'points' | 'previous_challenge_id' | 'flag' | 'is_active' | 'stack' | null>(null)
    const [editTitle, setEditTitle] = useState('')
    const [editDescription, setEditDescription] = useState('')
    const [editCategory, setEditCategory] = useState<string>(CHALLENGE_CATEGORIES[0])
    const [editPoints, setEditPoints] = useState(100)
    const [editPreviousChallengeId, setEditPreviousChallengeId] = useState<number | ''>('')
    const [editFlag, setEditFlag] = useState('')
    const [editIsActive, setEditIsActive] = useState(true)
    const [editStackEnabled, setEditStackEnabled] = useState(false)
    const portIdRef = useRef(0)
    const newPortRow = (port?: TargetPortSpec): TargetPortRow => ({
        id: `port-${portIdRef.current++}`,
        container_port: port?.container_port ?? 80,
        protocol: port?.protocol ?? 'TCP',
    })
    const [editStackTargetPorts, setEditStackTargetPorts] = useState<TargetPortRow[]>([newPortRow()])
    const [editStackPodSpec, setEditStackPodSpec] = useState('')
    const [loadedStackPodSpec, setLoadedStackPodSpec] = useState('')
    const [loadedPreviousChallengeId, setLoadedPreviousChallengeId] = useState<number | null>(null)
    const [editFile, setEditFile] = useState<File | null>(null)
    const [editFileError, setEditFileError] = useState('')
    const [editFileUploading, setEditFileUploading] = useState(false)
    const [editFileSuccess, setEditFileSuccess] = useState('')
    const readQueryState = (): { q: string; page: number; category: string; level: number; active: ActiveFilter; sort: SortFilter } => {
        if (typeof window === 'undefined') return { q: '', page: 1, category: 'all', level: 0, active: 'all' as ActiveFilter, sort: 'latest' as SortFilter }
        const params = new URLSearchParams(window.location.search)
        const parsedPage = Number(params.get('page'))
        const parsedLevel = Number(params.get('level'))
        const activeParam = params.get('active')
        const sortParam = params.get('sort')
        return {
            q: (params.get('q') ?? '').trim(),
            page: Number.isInteger(parsedPage) && parsedPage > 0 ? parsedPage : 1,
            category: (params.get('category') ?? 'all').trim() || 'all',
            level: Number.isInteger(parsedLevel) && parsedLevel >= 1 && parsedLevel <= 10 ? parsedLevel : 0,
            active: activeParam === 'active' || activeParam === 'inactive' ? activeParam : 'all',
            sort: sortParam === 'oldest' || sortParam === 'most_solved' || sortParam === 'least_solved' ? sortParam : 'latest',
        }
    }
    const initialQueryState = readQueryState()
    const [searchQuery, setSearchQuery] = useState(initialQueryState.q)
    const [appliedSearch, setAppliedSearch] = useState(initialQueryState.q)
    const [categoryFilter, setCategoryFilter] = useState(initialQueryState.category)
    const [levelFilter, setLevelFilter] = useState(initialQueryState.level)
    const [activeFilter, setActiveFilter] = useState<ActiveFilter>(initialQueryState.active)
    const [sortFilter, setSortFilter] = useState<SortFilter>(initialQueryState.sort)
    const [page, setPage] = useState(initialQueryState.page)
    const [pagination, setPagination] = useState({ page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
    const challengeLookup = new Map<number, Challenge>(challenges.map((item) => [item.id, item]))
    const formatChallengeOption = (item: Challenge) => {
        const categoryValue = 'category' in item && item.category ? item.category : t('common.na')
        return `#${item.id} ${item.title} (${t(getCategoryKey(categoryValue))})`
    }

    useEffect(() => {
        loadChallenges()
    }, [page, appliedSearch, categoryFilter, levelFilter, sortFilter])

    const pushQueryState = (next: { q: string; page: number; category: string; level: number; active: ActiveFilter; sort: SortFilter }) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams()
        if (next.q.trim() !== '') params.set('q', next.q.trim())
        if (next.page > 1) params.set('page', String(next.page))
        if (next.category !== 'all') params.set('category', next.category)
        if (next.level > 0) params.set('level', String(next.level))
        if (next.active !== 'all') params.set('active', next.active)
        if (next.sort !== 'latest') params.set('sort', next.sort)
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }

    useEffect(() => {
        const onPopState = () => {
            const state = readQueryState()
            setSearchQuery(state.q)
            setAppliedSearch(state.q)
            setCategoryFilter(state.category)
            setLevelFilter(state.level)
            setActiveFilter(state.active as ActiveFilter)
            setSortFilter(state.sort as SortFilter)
            setPage(state.page)
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    const loadChallenges = async () => {
        setLoading(true)
        setErrorMessage('')

        try {
            const response = await api.searchChallenges(appliedSearch, page, 20, {
                category: categoryFilter === 'all' ? undefined : categoryFilter,
                level: levelFilter > 0 ? levelFilter : undefined,
                sort: sortFilter,
            })
            const filtered = response.challenges.filter((challenge) => {
                if (activeFilter === 'active') return challenge.is_active !== false
                if (activeFilter === 'inactive') return challenge.is_active === false
                return true
            })
            setChallenges(filtered)
            setPagination(response.pagination)
        } catch (error) {
            const formatted = formatApiError(error, t)
            setErrorMessage(formatted.message)
            setPagination({ page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
        } finally {
            setLoading(false)
        }
    }

    const openEditor = async (challenge: Challenge) => {
        setManageFieldErrors({})
        setErrorMessage('')
        setSuccessMessage('')
        setEditFileError('')
        setEditFileSuccess('')
        setEditFile(null)
        setEditingField(null)
        setEditFlag('')

        if (expandedChallengeId === challenge.id) {
            setExpandedChallengeId(null)
            return
        }

        setExpandedChallengeId(challenge.id)
        setEditTitle(challenge.title)
        setEditDescription('description' in challenge ? challenge.description : '')
        setEditCategory('category' in challenge ? challenge.category : CHALLENGE_CATEGORIES[0])
        setEditPoints(challenge.points)
        setEditIsActive(challenge.is_active)
        setEditPreviousChallengeId('previous_challenge_id' in challenge && challenge.previous_challenge_id !== undefined ? (challenge.previous_challenge_id ?? '') : '')
        setLoadedPreviousChallengeId('previous_challenge_id' in challenge ? (challenge.previous_challenge_id ?? null) : null)
        setEditStackEnabled('stack_enabled' in challenge ? challenge.stack_enabled : false)
        const challengePorts = 'stack_target_ports' in challenge ? challenge.stack_target_ports : []
        setEditStackTargetPorts(Array.isArray(challengePorts) && challengePorts.length > 0 ? challengePorts.map((port) => newPortRow(port)) : [newPortRow()])
        setEditStackPodSpec('')
        setLoadedStackPodSpec('')

        try {
            setManageLoading(true)
            const detail = await api.adminChallenge(challenge.id)
            setEditTitle(detail.title)
            setEditDescription(detail.description)
            setEditCategory(detail.category)
            setEditPoints(detail.points)
            setEditIsActive(detail.is_active)
            setEditPreviousChallengeId(detail.previous_challenge_id ?? '')
            setLoadedPreviousChallengeId(detail.previous_challenge_id ?? null)
            setEditStackEnabled(detail.stack_enabled)
            setEditStackTargetPorts(detail.stack_target_ports && detail.stack_target_ports.length > 0 ? detail.stack_target_ports.map((port) => newPortRow(port)) : [newPortRow()])
            const podSpecValue = detail.stack_pod_spec ?? ''
            setEditStackPodSpec(podSpecValue)
            setLoadedStackPodSpec(podSpecValue)
            setChallenges((prev) => prev.map((item) => (item.id === detail.id ? detail : item)))
        } catch (error) {
            const formatted = formatApiError(error, t)
            setErrorMessage(formatted.message)
        } finally {
            setManageLoading(false)
        }
    }

    const beginEdit = (field: typeof editingField) => {
        setEditingField(field)
        setManageFieldErrors({})
        setErrorMessage('')
        setSuccessMessage('')
        if (field === 'flag') {
            setEditFlag('')
        }
    }

    const cancelEdit = (field: typeof editingField, challenge: Challenge) => {
        setEditingField(null)
        setManageFieldErrors({})
        const detail: ChallengeDetail | null = 'description' in challenge ? challenge : null
        if (field === 'title') setEditTitle(challenge.title)
        if (field === 'description') setEditDescription(detail?.description ?? '')
        if (field === 'category') setEditCategory(detail?.category ?? CHALLENGE_CATEGORIES[0])
        if (field === 'points') setEditPoints(detail?.points ?? challenge.points)
        if (field === 'previous_challenge_id') setEditPreviousChallengeId(loadedPreviousChallengeId ?? '')
        if (field === 'flag') setEditFlag('')
        if (field === 'is_active') setEditIsActive(detail?.is_active ?? true)
        if (field === 'stack') {
            setEditStackEnabled(detail?.stack_enabled ?? false)
            setEditStackTargetPorts(detail?.stack_target_ports && detail.stack_target_ports.length > 0 ? detail.stack_target_ports.map((port) => newPortRow(port)) : [newPortRow()])
            setEditStackPodSpec(loadedStackPodSpec)
        }
    }

    const saveField = async (challenge: Challenge, field: typeof editingField) => {
        setManageFieldErrors({})
        setErrorMessage('')
        setSuccessMessage('')

        if (!field) return

        const detail: ChallengeDetail | null = 'description' in challenge ? challenge : null
        const payload: ChallengeUpdatePayload = {}

        if (field === 'title') {
            if (editTitle === challenge.title) {
                setEditingField(null)
                return
            }
            payload.title = editTitle
        }

        if (field === 'description') {
            if (detail && editDescription === detail.description) {
                setEditingField(null)
                return
            }
            payload.description = editDescription
        }

        if (field === 'category') {
            if (detail && editCategory === detail.category) {
                setEditingField(null)
                return
            }
            payload.category = editCategory
        }

        if (field === 'points') {
            if (detail && Number(editPoints) === detail.points) {
                setEditingField(null)
                return
            }
            payload.points = Number(editPoints)
        }

        if (field === 'previous_challenge_id') {
            const nextValue = editPreviousChallengeId === '' ? null : Number(editPreviousChallengeId)
            const currentValue = loadedPreviousChallengeId
            if (nextValue === currentValue) {
                setEditingField(null)
                return
            }
            if (nextValue !== null && Number.isNaN(nextValue)) {
                setManageFieldErrors({ previous_challenge_id: t('errors.invalid') })
                return
            }
            payload.previous_challenge_id = nextValue
        }

        if (field === 'flag') {
            const trimmed = editFlag.trim()
            if (!trimmed) {
                setManageFieldErrors({ flag: t('errors.required') })
                return
            }
            payload.flag = trimmed
        }

        if (field === 'is_active') {
            if (detail && editIsActive === detail.is_active) {
                setEditingField(null)
                return
            }
            payload.is_active = editIsActive
        }

        if (field === 'stack') {
            const normalizePorts = (ports: TargetPortRow[] | TargetPortSpec[]) => ports.map((port) => ({ container_port: port.container_port, protocol: port.protocol }))
            const stackChanged = editStackEnabled !== (detail?.stack_enabled ?? false) || JSON.stringify(normalizePorts(editStackTargetPorts)) !== JSON.stringify(detail?.stack_target_ports ?? []) || editStackPodSpec !== loadedStackPodSpec

            if (!stackChanged) {
                setEditingField(null)
                return
            }

            payload.stack_enabled = editStackEnabled

            if (editStackEnabled) {
                if (!editStackPodSpec.trim()) {
                    setManageFieldErrors({ stack_pod_spec: t('errors.required') })
                    return
                }
                payload.stack_target_ports = editStackTargetPorts.map(({ container_port, protocol }) => ({
                    container_port,
                    protocol,
                }))
                payload.stack_pod_spec = editStackPodSpec
            }
        }

        setManageLoading(true)

        try {
            const updated = await api.updateChallenge(challenge.id, payload)

            setChallenges((prev) => prev.map((item) => (item.id === updated.id ? updated : item)))
            setSuccessMessage(t('admin.manage.successUpdated', { title: updated.title }))

            setEditTitle(updated.title)
            setEditDescription(updated.description)
            setEditCategory(updated.category)
            setEditPoints(updated.points)
            setEditIsActive(updated.is_active)
            setEditPreviousChallengeId(updated.previous_challenge_id ?? '')
            setLoadedPreviousChallengeId(updated.previous_challenge_id ?? null)
            setEditStackEnabled(updated.stack_enabled)
            setEditStackTargetPorts(updated.stack_target_ports && updated.stack_target_ports.length > 0 ? updated.stack_target_ports.map((port) => newPortRow(port)) : [newPortRow()])
            if (!updated.stack_enabled) {
                setEditStackPodSpec('')
                setLoadedStackPodSpec('')
            } else {
                setLoadedStackPodSpec(editStackPodSpec)
            }
            setEditFlag('')
            setEditingField(null)
        } catch (error) {
            const formatted = formatApiError(error, t)
            setErrorMessage(formatted.message)
            setManageFieldErrors(formatted.fieldErrors)
        } finally {
            setManageLoading(false)
        }
    }

    const uploadEditFile = async (challenge: Challenge) => {
        setEditFileError('')
        setEditFileSuccess('')

        if (!editFile) {
            setEditFileError(t('admin.manage.selectZip'))
            return
        }

        if (!isZipFile(editFile)) {
            setEditFileError(t('admin.create.onlyZip'))
            return
        }

        setEditFileUploading(true)

        try {
            const uploadResponse = await api.requestChallengeFileUpload(challenge.id, editFile.name)
            await uploadPresignedPost(uploadResponse.upload, editFile)
            setChallenges((prev) => prev.map((item) => (item.id === uploadResponse.challenge.id ? uploadResponse.challenge : item)))
            setEditFileSuccess(t('admin.manage.fileUploaded'))
            setEditFile(null)
        } catch (error) {
            const formatted = formatApiError(error, t)
            setEditFileError(formatted.message)
        } finally {
            setEditFileUploading(false)
        }
    }

    const deleteEditFile = async (challenge: Challenge) => {
        const confirmed = window.confirm(t('admin.manage.confirmDeleteFile', { title: challenge.title, id: challenge.id }))
        if (!confirmed) return

        setEditFileError('')
        setEditFileSuccess('')
        setEditFileUploading(true)

        try {
            const updated = await api.deleteChallengeFile(challenge.id)
            setChallenges((prev) => prev.map((item) => (item.id === updated.id ? updated : item)))
            setEditFileSuccess(t('admin.manage.fileDeleted'))
        } catch (error) {
            const formatted = formatApiError(error, t)
            setEditFileError(formatted.message)
        } finally {
            setEditFileUploading(false)
        }
    }

    const deleteChallenge = async (challenge: Challenge) => {
        const confirmed = window.confirm(t('admin.manage.confirmDeleteChallenge', { title: challenge.title, id: challenge.id }))
        if (!confirmed) return

        setManageLoading(true)
        setManageFieldErrors({})
        setErrorMessage('')
        setSuccessMessage('')

        try {
            await api.deleteChallenge(challenge.id)
            setChallenges((prev) => prev.filter((item) => item.id !== challenge.id))
            setSuccessMessage(t('admin.manage.successDeleted', { title: challenge.title }))
            if (expandedChallengeId === challenge.id) {
                setExpandedChallengeId(null)
            }
        } catch (error) {
            const formatted = formatApiError(error, t)
            setErrorMessage(formatted.message)
        } finally {
            setManageLoading(false)
        }
    }

    return (
        <div className='space-y-4'>
            <div className='flex items-center justify-between'>
                <button className='text-xs uppercase tracking-wide text-text-subtle hover:text-text cursor-pointer' onClick={loadChallenges} disabled={loading}>
                    {loading ? t('common.loading') : t('common.refresh')}
                </button>
            </div>

            <div className='space-y-2 bg-transparent shadow-none md:bg-surface md:p-3 dark:bg-surface'>
                <input
                    type='text'
                    placeholder={t('common.search')}
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    className='w-full rounded-lg border border-border/70 bg-surface px-4 py-2.5 text-sm text-text placeholder-text-subtle transition focus:border-accent focus:outline-none'
                />
                <div className='flex flex-wrap gap-2'>
                    <button
                        type='button'
                        className='rounded-md border border-border/70 bg-surface-muted px-4 py-2 text-sm text-text transition hover:bg-surface-subtle'
                        onClick={() => {
                            const nextQ = searchQuery.trim()
                            setAppliedSearch(nextQ)
                            setPage(1)
                            pushQueryState({ q: nextQ, page: 1, category: categoryFilter, level: levelFilter, active: activeFilter, sort: sortFilter })
                        }}
                    >
                        {t('common.search')}
                    </button>
                    <button
                        type='button'
                        className='rounded-md border border-border/70 bg-surface-muted px-4 py-2 text-sm text-text transition hover:bg-surface-subtle'
                        onClick={() => {
                            setSearchQuery('')
                            setAppliedSearch('')
                            setCategoryFilter('all')
                            setLevelFilter(0)
                            setActiveFilter('all')
                            setSortFilter('latest')
                            setPage(1)
                            pushQueryState({ q: '', page: 1, category: 'all', level: 0, active: 'all', sort: 'latest' })
                        }}
                    >
                        {t('common.reset')}
                    </button>
                </div>

                <div className='flex flex-wrap items-center gap-2'>
                    <span className='w-14 text-xs text-text-muted'>{t('challenges.filterCategory')}</span>
                    <button
                        type='button'
                        className={`rounded-md border px-3 py-1 text-xs ${categoryFilter === 'all' ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                        onClick={() => {
                            setCategoryFilter('all')
                            setPage(1)
                            pushQueryState({ q: appliedSearch, page: 1, category: 'all', level: levelFilter, active: activeFilter, sort: sortFilter })
                        }}
                    >
                        {t('common.all')}
                    </button>
                    {CHALLENGE_CATEGORIES.map((category) => (
                        <button
                            key={category}
                            type='button'
                            className={`rounded-md border px-3 py-1 text-xs ${categoryFilter === category ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                            onClick={() => {
                                setCategoryFilter(category)
                                setPage(1)
                                pushQueryState({ q: appliedSearch, page: 1, category, level: levelFilter, active: activeFilter, sort: sortFilter })
                            }}
                        >
                            {t(getCategoryKey(category))}
                        </button>
                    ))}
                </div>

                <div className='flex flex-wrap items-center gap-2'>
                    <span className='w-14 text-xs text-text-muted'>{t('challenges.filterLevel')}</span>
                    <button
                        type='button'
                        className={`rounded-md border px-3 py-1 text-xs ${levelFilter === 0 ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                        onClick={() => {
                            setLevelFilter(0)
                            setPage(1)
                            pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: 0, active: activeFilter, sort: sortFilter })
                        }}
                    >
                        {t('common.all')}
                    </button>
                    {Array.from({ length: 10 }, (_, idx) => idx + 1).map((level) => (
                        <button
                            key={level}
                            type='button'
                            className={`rounded-md border px-3 py-1 text-xs ${levelFilter === level ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                            onClick={() => {
                                setLevelFilter(level)
                                setPage(1)
                                pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level, active: activeFilter, sort: sortFilter })
                            }}
                        >
                            {level}
                        </button>
                    ))}
                </div>

                <div className='flex flex-wrap items-center gap-2'>
                    <span className='w-14 text-xs text-text-muted'>{t('common.status')}</span>
                    {(['all', 'active', 'inactive'] as const).map((key) => (
                        <button
                            key={key}
                            type='button'
                            className={`rounded-md border px-3 py-1 text-xs ${activeFilter === key ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                            onClick={() => {
                                setActiveFilter(key)
                                setPage(1)
                                pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: levelFilter, active: key, sort: sortFilter })
                            }}
                        >
                            {key === 'all' ? t('common.all') : key === 'active' ? t('common.active') : t('common.inactive')}
                        </button>
                    ))}
                </div>

                <div className='flex flex-wrap items-center gap-2'>
                    <span className='w-14 text-xs text-text-muted'>{t('challenges.filterSort')}</span>
                    {(['latest', 'oldest', 'most_solved', 'least_solved'] as const).map((key) => (
                        <button
                            key={key}
                            type='button'
                            className={`rounded-md border px-3 py-1 text-xs ${sortFilter === key ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                            onClick={() => {
                                setSortFilter(key)
                                setPage(1)
                                pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: levelFilter, active: activeFilter, sort: key })
                            }}
                        >
                            {t(`challenges.sort.${key}`)}
                        </button>
                    ))}
                </div>
            </div>

            {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
            {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

            {loading ? (
                <p className='text-sm text-text-subtle'>{t('admin.manage.loadingChallenges')}</p>
            ) : (
                <div className='-mx-4 md:mx-0 overflow-visible md:overflow-hidden rounded-none md:rounded-xl bg-transparent md:bg-surface md:shadow-sm'>
                    <div className='overflow-x-auto'>
                        <div className='min-w-[980px]'>
                            <div className='grid min-w-[980px] grid-cols-[72px_minmax(0,1fr)_140px_90px_100px_100px_110px_130px] bg-surface-muted px-4 py-3 text-[12px] text-text-muted'>
                                <p className='font-medium'>{t('common.id')}</p>
                                <p className='font-medium'>{t('common.title')}</p>
                                <p className='font-medium'>{t('common.category')}</p>
                                <p className='font-medium'>LEVEL</p>
                                <p className='font-medium'>{t('common.points')}</p>
                                <p className='font-medium'>{t('challenges.tableSolveCount')}</p>
                                <p className='font-medium'>{t('common.status')}</p>
                                <p className='text-right font-medium'>{t('common.action')}</p>
                            </div>
                            {challenges.map((challenge) => {
                                const isActive = 'is_active' in challenge ? challenge.is_active !== false : true
                                const categoryLabel = 'category' in challenge ? t(getCategoryKey(challenge.category)) : t('common.na')
                                const solveCount = challenge.solve_count
                                const hasFile = 'has_file' in challenge && challenge.has_file
                                const fileName = 'file_name' in challenge ? challenge.file_name : null

                                return (
                                    <Fragment key={challenge.id}>
                                        <div className='grid min-w-[980px] grid-cols-[72px_minmax(0,1fr)_140px_90px_100px_100px_110px_130px] items-center px-4 py-4 transition hover:bg-surface-muted/40'>
                                            <p className='whitespace-nowrap text-sm text-text'>{challenge.id}</p>
                                            <p className='truncate pr-3 text-sm text-text'>{challenge.title}</p>
                                            <p className='text-sm text-text'>{categoryLabel}</p>
                                            <p className='text-sm text-text'>{challenge.level}</p>
                                            <p className='text-sm text-text'>{challenge.points}</p>
                                            <p className='text-sm text-text'>{solveCount}</p>
                                            <p className='text-sm text-text-muted'>{isActive ? t('admin.manage.statusActive') : t('admin.manage.statusInactive')}</p>
                                            <div className='whitespace-nowrap text-right text-sm'>
                                                <div className='flex items-center justify-end gap-3'>
                                                    <button className='text-accent hover:text-accent-strong cursor-pointer' onClick={() => openEditor(challenge)} disabled={manageLoading}>
                                                        {expandedChallengeId === challenge.id ? t('admin.manage.closeEdit') : t('admin.manage.edit')}
                                                    </button>
                                                    <button className='text-danger hover:text-danger-strong cursor-pointer' onClick={() => deleteChallenge(challenge)} disabled={manageLoading}>
                                                        {t('admin.manage.delete')}
                                                    </button>
                                                </div>
                                            </div>
                                        </div>
                                        {expandedChallengeId === challenge.id ? (
                                            <div className='bg-surface/70 px-6 py-6'>
                                                <div className='space-y-5'>
                                                    <div>
                                                        <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor={`manage-title-${challenge.id}`}>
                                                            {t('common.title')}
                                                        </label>
                                                        {editingField === 'title' ? (
                                                            <div className='mt-2 space-y-2'>
                                                                <input
                                                                    id={`manage-title-${challenge.id}`}
                                                                    className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                                    type='text'
                                                                    value={editTitle}
                                                                    onChange={(event) => setEditTitle(event.target.value)}
                                                                    disabled={manageLoading}
                                                                />
                                                                <div className='flex flex-wrap items-center gap-3'>
                                                                    <button
                                                                        className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => saveField(challenge, 'title')}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                    </button>
                                                                    <button
                                                                        className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => cancelEdit('title', challenge)}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {t('common.cancel')}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                        ) : (
                                                            <div className='mt-2 flex items-center justify-between gap-4 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text'>
                                                                <span>{editTitle}</span>
                                                                <button
                                                                    className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                    type='button'
                                                                    onClick={() => beginEdit('title')}
                                                                    disabled={manageLoading || editingField !== null}
                                                                >
                                                                    {t('common.edit')}
                                                                </button>
                                                            </div>
                                                        )}
                                                        {manageFieldErrors.title ? (
                                                            <p className='mt-2 text-xs text-danger'>
                                                                {t('common.title')}: {manageFieldErrors.title}
                                                            </p>
                                                        ) : null}
                                                    </div>
                                                    <div>
                                                        <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor={`manage-description-${challenge.id}`}>
                                                            {t('common.description')}
                                                        </label>
                                                        {editingField === 'description' ? (
                                                            <div className='mt-2 space-y-2'>
                                                                <div className='w-full rounded-xl border border-border bg-surface py-4 text-sm text-text focus-within:border-accent'>
                                                                    <MonacoEditor value={editDescription} onChange={setEditDescription} />
                                                                </div>

                                                                <div className='flex flex-wrap items-center gap-3'>
                                                                    <button
                                                                        className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => saveField(challenge, 'description')}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                    </button>
                                                                    <button
                                                                        className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => cancelEdit('description', challenge)}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {t('common.cancel')}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                        ) : (
                                                            <div className='mt-2 flex items-start justify-between gap-4 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text'>
                                                                <p className='whitespace-pre-wrap'>{editDescription}</p>
                                                                <button
                                                                    className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                    type='button'
                                                                    onClick={() => beginEdit('description')}
                                                                    disabled={manageLoading || editingField !== null}
                                                                >
                                                                    {t('common.edit')}
                                                                </button>
                                                            </div>
                                                        )}
                                                        {manageFieldErrors.description ? (
                                                            <p className='mt-2 text-xs text-danger'>
                                                                {t('common.description')}: {manageFieldErrors.description}
                                                            </p>
                                                        ) : null}
                                                    </div>
                                                    <div className='grid gap-4 md:grid-cols-4'>
                                                        <div>
                                                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor={`manage-category-${challenge.id}`}>
                                                                {t('common.category')}
                                                            </label>
                                                            {editingField === 'category' ? (
                                                                <div className='mt-2 space-y-2'>
                                                                    <select
                                                                        id={`manage-category-${challenge.id}`}
                                                                        className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                                        value={editCategory}
                                                                        onChange={(event) => setEditCategory(event.target.value)}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {CHALLENGE_CATEGORIES.map((option) => (
                                                                            <option key={option} value={option}>
                                                                                {t(getCategoryKey(option))}
                                                                            </option>
                                                                        ))}
                                                                    </select>
                                                                    <div className='flex flex-wrap items-center gap-3'>
                                                                        <button
                                                                            className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                            type='button'
                                                                            onClick={() => saveField(challenge, 'category')}
                                                                            disabled={manageLoading}
                                                                        >
                                                                            {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                        </button>
                                                                        <button
                                                                            className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                            type='button'
                                                                            onClick={() => cancelEdit('category', challenge)}
                                                                            disabled={manageLoading}
                                                                        >
                                                                            {t('common.cancel')}
                                                                        </button>
                                                                    </div>
                                                                </div>
                                                            ) : (
                                                                <div className='mt-2 flex items-center justify-between gap-4 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text'>
                                                                    <span>{t(getCategoryKey(editCategory))}</span>
                                                                    <button
                                                                        className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                        type='button'
                                                                        onClick={() => beginEdit('category')}
                                                                        disabled={manageLoading || editingField !== null}
                                                                    >
                                                                        {t('common.edit')}
                                                                    </button>
                                                                </div>
                                                            )}
                                                            {manageFieldErrors.category ? (
                                                                <p className='mt-2 text-xs text-danger'>
                                                                    {t('common.category')}: {manageFieldErrors.category}
                                                                </p>
                                                            ) : null}
                                                        </div>
                                                        <div>
                                                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor={`manage-points-${challenge.id}`}>
                                                                {t('common.points')}
                                                            </label>
                                                            {editingField === 'points' ? (
                                                                <div className='mt-2 space-y-2'>
                                                                    <input
                                                                        id={`manage-points-${challenge.id}`}
                                                                        className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                                        type='number'
                                                                        min={0}
                                                                        value={editPoints}
                                                                        onChange={(event) => setEditPoints(Number(event.target.value))}
                                                                        disabled={manageLoading}
                                                                    />
                                                                    <div className='flex flex-wrap items-center gap-3'>
                                                                        <button
                                                                            className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                            type='button'
                                                                            onClick={() => saveField(challenge, 'points')}
                                                                            disabled={manageLoading}
                                                                        >
                                                                            {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                        </button>
                                                                        <button
                                                                            className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                            type='button'
                                                                            onClick={() => cancelEdit('points', challenge)}
                                                                            disabled={manageLoading}
                                                                        >
                                                                            {t('common.cancel')}
                                                                        </button>
                                                                    </div>
                                                                </div>
                                                            ) : (
                                                                <div className='mt-2 flex items-center justify-between gap-4 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text'>
                                                                    <span>{editPoints}</span>
                                                                    <button
                                                                        className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                        type='button'
                                                                        onClick={() => beginEdit('points')}
                                                                        disabled={manageLoading || editingField !== null}
                                                                    >
                                                                        {t('common.edit')}
                                                                    </button>
                                                                </div>
                                                            )}
                                                            {manageFieldErrors.points ? (
                                                                <p className='mt-2 text-xs text-danger'>
                                                                    {t('common.points')}: {manageFieldErrors.points}
                                                                </p>
                                                            ) : null}
                                                        </div>
                                                        <div>
                                                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor={`manage-previous-challenge-${challenge.id}`}>
                                                                {t('admin.create.previousChallenge')}
                                                            </label>
                                                            {editingField === 'previous_challenge_id' ? (
                                                                <div className='mt-2 space-y-2'>
                                                                    <select
                                                                        id={`manage-previous-challenge-${challenge.id}`}
                                                                        className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                                        value={editPreviousChallengeId === '' ? '' : String(editPreviousChallengeId)}
                                                                        onChange={(event) => {
                                                                            const value = event.target.value
                                                                            setEditPreviousChallengeId(value === '' ? '' : Number(value))
                                                                        }}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        <option value=''>{t('admin.create.previousChallengeNone')}</option>
                                                                        {challenges
                                                                            .filter((item) => item.id !== challenge.id)
                                                                            .map((item) => (
                                                                                <option key={item.id} value={item.id}>
                                                                                    {formatChallengeOption(item)}
                                                                                </option>
                                                                            ))}
                                                                    </select>
                                                                    <div className='flex flex-wrap items-center gap-3'>
                                                                        <button
                                                                            className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                            type='button'
                                                                            onClick={() => saveField(challenge, 'previous_challenge_id')}
                                                                            disabled={manageLoading}
                                                                        >
                                                                            {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                        </button>
                                                                        <button
                                                                            className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                            type='button'
                                                                            onClick={() => cancelEdit('previous_challenge_id', challenge)}
                                                                            disabled={manageLoading}
                                                                        >
                                                                            {t('common.cancel')}
                                                                        </button>
                                                                    </div>
                                                                </div>
                                                            ) : (
                                                                <div className='mt-2 flex items-center justify-between gap-4 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text'>
                                                                    <span>
                                                                        {editPreviousChallengeId === ''
                                                                            ? t('admin.create.previousChallengeNone')
                                                                            : (() => {
                                                                                  const found = challengeLookup.get(Number(editPreviousChallengeId))
                                                                                  if (found) return formatChallengeOption(found)
                                                                                  return `#${editPreviousChallengeId} ${t('common.na')}`
                                                                              })()}
                                                                    </span>
                                                                    <button
                                                                        className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                        type='button'
                                                                        onClick={() => beginEdit('previous_challenge_id')}
                                                                        disabled={manageLoading || editingField !== null}
                                                                    >
                                                                        {t('common.edit')}
                                                                    </button>
                                                                </div>
                                                            )}
                                                            {manageFieldErrors.previous_challenge_id ? (
                                                                <p className='mt-2 text-xs text-danger'>
                                                                    {t('admin.create.previousChallenge')}: {manageFieldErrors.previous_challenge_id}
                                                                </p>
                                                            ) : null}
                                                        </div>
                                                    </div>
                                                    <div>
                                                        <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor={`manage-flag-${challenge.id}`}>
                                                            {t('common.flag')}
                                                        </label>
                                                        {editingField === 'flag' ? (
                                                            <div className='mt-2 space-y-2'>
                                                                <input
                                                                    id={`manage-flag-${challenge.id}`}
                                                                    className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                                    type='password'
                                                                    value={editFlag}
                                                                    onChange={(event) => setEditFlag(event.target.value)}
                                                                    placeholder={t('admin.manage.flagPlaceholder')}
                                                                    disabled={manageLoading}
                                                                />
                                                                <div className='flex flex-wrap items-center gap-3'>
                                                                    <button
                                                                        className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => saveField(challenge, 'flag')}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                    </button>
                                                                    <button
                                                                        className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => cancelEdit('flag', challenge)}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {t('common.cancel')}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                        ) : (
                                                            <div className='mt-2 flex items-center justify-between gap-4 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text'>
                                                                <span>{t('admin.manage.flagMasked')}</span>
                                                                <button
                                                                    className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                    type='button'
                                                                    onClick={() => beginEdit('flag')}
                                                                    disabled={manageLoading || editingField !== null}
                                                                >
                                                                    {t('common.edit')}
                                                                </button>
                                                            </div>
                                                        )}
                                                        {manageFieldErrors.flag ? (
                                                            <p className='mt-2 text-xs text-danger'>
                                                                {t('common.flag')}: {manageFieldErrors.flag}
                                                            </p>
                                                        ) : null}
                                                        <p className='mt-2 text-xs text-text-subtle'>{t('admin.manage.flagHint')}</p>
                                                    </div>
                                                    <div>
                                                        <label className='text-xs uppercase tracking-wide text-text-muted'>{t('common.active')}</label>
                                                        {editingField === 'is_active' ? (
                                                            <div className='mt-2 space-y-2'>
                                                                <label className='flex items-center gap-3 text-sm text-text'>
                                                                    <input
                                                                        type='checkbox'
                                                                        checked={editIsActive}
                                                                        onChange={(event) => setEditIsActive(event.target.checked)}
                                                                        className='h-4 w-4 rounded border-border'
                                                                        disabled={manageLoading}
                                                                    />
                                                                    {editIsActive ? t('admin.manage.statusActive') : t('admin.manage.statusInactive')}
                                                                </label>
                                                                <div className='flex flex-wrap items-center gap-3'>
                                                                    <button
                                                                        className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => saveField(challenge, 'is_active')}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                    </button>
                                                                    <button
                                                                        className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => cancelEdit('is_active', challenge)}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {t('common.cancel')}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                        ) : (
                                                            <div className='mt-2 flex items-center justify-between gap-4 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text'>
                                                                <span>{editIsActive ? t('admin.manage.statusActive') : t('admin.manage.statusInactive')}</span>
                                                                <button
                                                                    className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                    type='button'
                                                                    onClick={() => beginEdit('is_active')}
                                                                    disabled={manageLoading || editingField !== null}
                                                                >
                                                                    {t('common.edit')}
                                                                </button>
                                                            </div>
                                                        )}
                                                    </div>
                                                    <div className='rounded-2xl border border-border bg-surface/60 p-4'>
                                                        <div className='flex items-center justify-between gap-4'>
                                                            <p className='text-xs uppercase tracking-wide text-text-subtle'>{t('admin.create.provideStack')}</p>
                                                            {editingField !== 'stack' ? (
                                                                <button
                                                                    className='text-xs text-accent hover:underline cursor-pointer disabled:opacity-60'
                                                                    type='button'
                                                                    onClick={() => beginEdit('stack')}
                                                                    disabled={manageLoading || editingField !== null}
                                                                >
                                                                    {t('common.edit')}
                                                                </button>
                                                            ) : null}
                                                        </div>
                                                        {editingField === 'stack' ? (
                                                            <div className='mt-3 space-y-3'>
                                                                <label className='flex items-center gap-3 text-sm text-text'>
                                                                    <input
                                                                        type='checkbox'
                                                                        checked={editStackEnabled}
                                                                        onChange={(event) => setEditStackEnabled(event.target.checked)}
                                                                        className='h-4 w-4 rounded border-border'
                                                                        disabled={manageLoading}
                                                                    />
                                                                    {editStackEnabled ? t('common.active') : t('common.inactive')}
                                                                </label>
                                                                {editStackEnabled ? (
                                                                    <div className='grid gap-4'>
                                                                        <div>
                                                                            <div className='flex flex-wrap items-center justify-between gap-2'>
                                                                                <label className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.create.targetPorts')}</label>
                                                                                <button
                                                                                    className='text-xs text-accent hover:underline disabled:opacity-60 cursor-pointer'
                                                                                    type='button'
                                                                                    onClick={() => setEditStackTargetPorts((prev) => (prev.length >= 24 ? prev : [...prev, newPortRow()]))}
                                                                                    disabled={manageLoading || editStackTargetPorts.length >= 24}
                                                                                >
                                                                                    {t('common.add')}
                                                                                </button>
                                                                            </div>
                                                                            <div className='mt-3 grid gap-3'>
                                                                                {editStackTargetPorts.map((port, index) => (
                                                                                    <div key={port.id} className='grid gap-3 sm:grid-cols-[1fr_120px_auto] items-center'>
                                                                                        <input
                                                                                            className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                                                            type='number'
                                                                                            min={1}
                                                                                            max={65535}
                                                                                            value={port.container_port}
                                                                                            onChange={(event) => {
                                                                                                const value = Number(event.target.value)
                                                                                                setEditStackTargetPorts((prev) =>
                                                                                                    prev.map((item, idx) =>
                                                                                                        idx === index
                                                                                                            ? {
                                                                                                                  ...item,
                                                                                                                  container_port: value,
                                                                                                              }
                                                                                                            : item,
                                                                                                    ),
                                                                                                )
                                                                                            }}
                                                                                            disabled={manageLoading}
                                                                                        />
                                                                                        <select
                                                                                            className='w-full min-w-22.5 rounded-xl border border-border bg-surface px-3 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                                                            value={port.protocol}
                                                                                            onChange={(event) => {
                                                                                                const value = event.target.value as TargetPortSpec['protocol']
                                                                                                setEditStackTargetPorts((prev) =>
                                                                                                    prev.map((item, idx) =>
                                                                                                        idx === index
                                                                                                            ? {
                                                                                                                  ...item,
                                                                                                                  protocol: value,
                                                                                                              }
                                                                                                            : item,
                                                                                                    ),
                                                                                                )
                                                                                            }}
                                                                                            disabled={manageLoading}
                                                                                        >
                                                                                            <option value='TCP'>TCP</option>
                                                                                            <option value='UDP'>UDP</option>
                                                                                        </select>
                                                                                        <button
                                                                                            className='min-w-18 rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                                            type='button'
                                                                                            onClick={() => setEditStackTargetPorts((prev) => prev.filter((_, idx) => idx !== index))}
                                                                                            disabled={manageLoading || editStackTargetPorts.length <= 1}
                                                                                        >
                                                                                            {t('common.remove')}
                                                                                        </button>
                                                                                    </div>
                                                                                ))}
                                                                            </div>
                                                                            {manageFieldErrors.stack_target_ports ? (
                                                                                <p className='mt-2 text-xs text-danger'>
                                                                                    {t('admin.create.targetPorts')}: {manageFieldErrors.stack_target_ports}
                                                                                </p>
                                                                            ) : null}
                                                                            {editStackTargetPorts.length >= 24 ? <p className='mt-2 text-xs text-text-muted'>{t('admin.create.maxPorts')}</p> : null}
                                                                        </div>
                                                                        <div>
                                                                            <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.create.podSpec')}</p>
                                                                            <div className='mt-2 w-full rounded-xl border border-border bg-surface py-4 text-sm text-text focus-within:border-accent'>
                                                                                <MonacoEditor language='yaml' value={editStackPodSpec} onChange={(value) => setEditStackPodSpec(value)} readonly={manageLoading} />
                                                                            </div>
                                                                            {manageFieldErrors.stack_pod_spec ? (
                                                                                <p className='mt-2 text-xs text-danger'>
                                                                                    {t('admin.create.podSpec')}: {manageFieldErrors.stack_pod_spec}
                                                                                </p>
                                                                            ) : null}
                                                                        </div>
                                                                    </div>
                                                                ) : null}
                                                                <div className='flex flex-wrap items-center gap-3'>
                                                                    <button
                                                                        className='rounded-lg bg-accent px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => saveField(challenge, 'stack')}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {manageLoading ? t('admin.site.saving') : t('common.save')}
                                                                    </button>
                                                                    <button
                                                                        className='rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                                        type='button'
                                                                        onClick={() => cancelEdit('stack', challenge)}
                                                                        disabled={manageLoading}
                                                                    >
                                                                        {t('common.cancel')}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                        ) : (
                                                            <div className='mt-3 space-y-1 text-sm text-text'>
                                                                <p>{editStackEnabled ? t('common.active') : t('common.inactive')}</p>
                                                                {editStackEnabled ? (
                                                                    <>
                                                                        <p>
                                                                            {t('admin.create.targetPorts')}:{' '}
                                                                            {editStackTargetPorts.length > 0 ? editStackTargetPorts.map((port) => `${port.container_port}/${port.protocol}`).join(', ') : t('common.pending')}
                                                                        </p>
                                                                        <p>
                                                                            {t('admin.create.podSpec')}: {loadedStackPodSpec ? t('admin.manage.podSpecConfigured') : t('admin.manage.podSpecMissing')}
                                                                        </p>
                                                                    </>
                                                                ) : null}
                                                            </div>
                                                        )}
                                                    </div>

                                                    <div className='rounded-xl border border-border bg-surface/60 p-4 text-sm text-text'>
                                                        <p className='text-xs uppercase tracking-wide text-text-subtle'>{t('admin.manage.challengeFile')}</p>
                                                        <p className='mt-2 text-sm text-text'>{hasFile ? (fileName ?? 'challenge.zip') : t('admin.manage.noFileUploaded')}</p>
                                                        <div className='mt-3 flex flex-wrap items-center gap-3'>
                                                            <input
                                                                className='w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text sm:w-auto'
                                                                type='file'
                                                                accept='.zip'
                                                                onChange={(event) => {
                                                                    const target = event.currentTarget
                                                                    setEditFile(target.files?.[0] ?? null)
                                                                    setEditFileError('')
                                                                    setEditFileSuccess('')
                                                                }}
                                                            />
                                                            <button
                                                                className='rounded-lg bg-contrast px-4 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-contrast/80 disabled:opacity-60 cursor-pointer'
                                                                type='button'
                                                                onClick={() => uploadEditFile(challenge)}
                                                                disabled={editFileUploading || manageLoading}
                                                            >
                                                                {editFileUploading ? t('admin.create.uploading') : t('admin.manage.uploadZip')}
                                                            </button>
                                                            {hasFile ? (
                                                                <button
                                                                    className='rounded-lg border border-danger/30 px-4 py-2 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60 cursor-pointer'
                                                                    type='button'
                                                                    onClick={() => deleteEditFile(challenge)}
                                                                    disabled={editFileUploading || manageLoading}
                                                                >
                                                                    {t('admin.manage.deleteFile')}
                                                                </button>
                                                            ) : null}
                                                        </div>
                                                        {editFileError ? <FormMessage variant='error' message={editFileError} className='mt-2' /> : null}
                                                        {editFileSuccess ? <FormMessage variant='success' message={editFileSuccess} className='mt-2' /> : null}
                                                    </div>

                                                    <div className='flex flex-col gap-3 sm:flex-row sm:justify-end'>
                                                        <button
                                                            className='rounded-xl border border-border px-5 py-3 text-sm text-text transition hover:border-border hover:text-text disabled:opacity-60 cursor-pointer'
                                                            type='button'
                                                            onClick={() => setExpandedChallengeId(null)}
                                                            disabled={manageLoading}
                                                        >
                                                            {t('common.cancel')}
                                                        </button>
                                                    </div>
                                                </div>
                                            </div>
                                        ) : null}
                                    </Fragment>
                                )
                            })}
                            {challenges.length === 0 ? <p className='px-6 py-8 text-center text-sm text-text-muted'>{t('admin.manage.noChallenges')}</p> : null}
                        </div>
                    </div>
                </div>
            )}
            <div className='mt-3 flex items-center justify-end gap-2 px-1 text-xs text-text-muted'>
                <button
                    type='button'
                    className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:cursor-not-allowed disabled:opacity-50'
                    disabled={!pagination.has_prev}
                    onClick={() => {
                        const nextPage = Math.max(1, page - 1)
                        setPage(nextPage)
                        pushQueryState({ q: appliedSearch, page: nextPage, category: categoryFilter, level: levelFilter, active: activeFilter, sort: sortFilter })
                    }}
                >
                    {t('common.previous')}
                </button>
                <span className='text-xs text-text-muted'>
                    {pagination.page} / {pagination.total_pages || 1}
                </span>
                <button
                    type='button'
                    className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:cursor-not-allowed disabled:opacity-50'
                    disabled={!pagination.has_next}
                    onClick={() => {
                        const nextPage = page + 1
                        setPage(nextPage)
                        pushQueryState({ q: appliedSearch, page: nextPage, category: categoryFilter, level: levelFilter, active: activeFilter, sort: sortFilter })
                    }}
                >
                    {t('common.next')}
                </button>
            </div>
        </div>
    )
}

export default ChallengeManagement
