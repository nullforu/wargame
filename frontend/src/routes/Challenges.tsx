import { useEffect, useMemo, useState } from 'react'
import { formatApiError } from '../lib/utils'
import type { Challenge } from '../lib/types'
import ChallengeModal from '../components/ChallengeModal'
import ChallengesView from '../components/ChallengesView'
import LoginRequired from '../components/LoginRequired'
import { getCategoryKey, useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'
import { CHALLENGE_CATEGORIES } from '../lib/constants'
import { useAuth } from '../lib/auth'

interface RouteProps {
    routeParams?: Record<string, string>
}

const CATEGORY_SET = new Set<string>(CHALLENGE_CATEGORIES)
const GROUP_BY_CATEGORY_STORAGE_KEY = 'wargame.challenges.groupByCategory'

const loadGroupByCategory = () => {
    if (typeof localStorage === 'undefined') return true

    const saved = localStorage.getItem(GROUP_BY_CATEGORY_STORAGE_KEY)
    if (saved === null) return true

    return saved === 'true'
}

const persistGroupByCategory = (value: boolean) => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem(GROUP_BY_CATEGORY_STORAGE_KEY, String(value))
    }
}

const Challenges = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const { state: auth } = useAuth()
    const [challenges, setChallenges] = useState<Challenge[]>([])
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')
    const [solvedIds, setSolvedIds] = useState<Set<number>>(new Set())
    const [selectedChallenge, setSelectedChallenge] = useState<Challenge | null>(null)
    const [groupByCategory, setGroupByCategory] = useState<boolean>(() => loadGroupByCategory())

    const activeChallenges = useMemo(() => challenges.filter((challenge) => ('is_active' in challenge ? challenge.is_active !== false : true)), [challenges])
    const inactiveChallenges = useMemo(() => challenges.filter((challenge) => ('is_active' in challenge ? challenge.is_active === false : false)), [challenges])
    const solvedCount = useMemo(() => solvedIds.size, [solvedIds])

    const challengesByCategory = useMemo(() => {
        const grouped = new Map<string, Challenge[]>()
        for (const challenge of challenges) {
            const category = 'category' in challenge && challenge.category ? challenge.category : t('common.na')
            const existing = grouped.get(category) ?? []
            existing.push(challenge)
            grouped.set(category, existing)
        }

        return grouped
    }, [challenges, t])

    const orderedCategories = useMemo(() => {
        const present = new Set(challengesByCategory.keys())
        const ordered = CHALLENGE_CATEGORIES.filter((category) => present.has(category))
        const extras = [...present].filter((category) => !CATEGORY_SET.has(category))

        return [...ordered, ...extras]
    }, [challengesByCategory])

    const loadChallenges = async () => {
        setLoading(true)
        setErrorMessage('')

        try {
            const data = await api.challenges()
            setChallenges(data.challenges)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setLoading(false)
        }
    }

    const loadSolved = async () => {
        try {
            if (!auth.user) {
                setSolvedIds(new Set())
                return
            }

            const userSolved = await api.userSolved(auth.user.id)
            setSolvedIds(new Set(userSolved.map((item) => item.challenge_id)))
        } catch {
            setSolvedIds(new Set())
        }
    }

    useEffect(() => {
        if (!auth.user) return
        void Promise.all([loadChallenges(), loadSolved()])
    }, [auth.user?.id])

    useEffect(() => {
        persistGroupByCategory(groupByCategory)
    }, [groupByCategory])

    const solvedSummary = t('challenges.solvedSummary', { solved: solvedCount, total: activeChallenges.length })
    const inactiveSummary = inactiveChallenges.length > 0 ? t('challenges.inactiveCount', { count: inactiveChallenges.length }) : ''
    const summaryText = [solvedSummary, inactiveSummary].filter(Boolean).join(' ')
    const stackSummaryText = auth.user && auth.user.stack_limit > 0 ? t('challenges.stackSummary', { count: auth.user.stack_count, limit: auth.user.stack_limit }) : ''

    const groupedCategories = useMemo(
        () =>
            orderedCategories.map((category) => ({
                id: category,
                label: t(getCategoryKey(category)),
                items: challengesByCategory.get(category) ?? [],
            })),
        [orderedCategories, challengesByCategory, t],
    )

    if (!auth.user) {
        return <LoginRequired title={t('challenges.title')} />
    }

    return (
        <section className='animate'>
            <ChallengesView
                title={t('challenges.title')}
                summaryText={summaryText}
                stackSummaryText={stackSummaryText}
                showSummary={true}
                groupByCategory={groupByCategory}
                toggleLabel={t('challenges.groupByCategory')}
                onGroupByCategoryChange={setGroupByCategory}
                loading={loading}
                loadingText={t('challenges.loading')}
                errorMessage={errorMessage}
                notStarted={false}
                notStartedText={t('challenges.notStarted')}
                startAtLabel={t('challenges.startAt')}
                startAtValue=''
                endAtLabel={t('challenges.endAt')}
                endAtValue=''
                ended={false}
                endedText={t('challenges.ended')}
                challenges={challenges}
                groupedCategories={groupedCategories}
                solvedIds={solvedIds}
                onSelectChallenge={setSelectedChallenge}
            />

            {selectedChallenge ? <ChallengeModal challenge={selectedChallenge} isSolved={solvedIds.has(selectedChallenge.id)} onClose={() => setSelectedChallenge(null)} onSolved={loadSolved} /> : null}
        </section>
    )
}

export default Challenges
