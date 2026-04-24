import { useEffect, useMemo, useState } from 'react'
import CreateChallenge from './admin/CreateChallenge'
import ChallengeManagement from './admin/ChallengeManagement'
import Users from './admin/Users'
import Stacks from './admin/Stacks'
import Affiliations from './admin/Affiliations'
import { useT } from '../lib/i18n'
import { useAuth } from '../lib/auth'

interface RouteProps {
    routeParams?: Record<string, string>
}

type AdminTabId = 'challenge_create' | 'challenge_management' | 'users' | 'stacks' | 'affiliations'
const TAB_PARAM = 'tab'
const ADMIN_TAB_IDS: AdminTabId[] = ['challenge_create', 'challenge_management', 'users', 'stacks', 'affiliations']

const getTabFromUrl = (): AdminTabId | null => {
    const params = new URLSearchParams(window.location.search)
    const value = params.get(TAB_PARAM)
    return ADMIN_TAB_IDS.includes(value as AdminTabId) ? (value as AdminTabId) : null
}

const Admin = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const { state: auth } = useAuth()
    const adminTabs = useMemo(
        () => [
            { id: 'challenge_create', label: t('admin.tab.createChallenge') },
            { id: 'challenge_management', label: t('admin.tab.challengeManagement') },
            { id: 'users', label: t('admin.tab.users') },
            { id: 'stacks', label: t('admin.tab.stacks') },
            { id: 'affiliations', label: t('admin.tab.affiliations') },
        ],
        [t],
    )

    const [activeTab, setActiveTab] = useState<AdminTabId>(() => getTabFromUrl() ?? 'challenge_create')

    useEffect(() => {
        const handlePopState = () => {
            const nextTab = getTabFromUrl()
            if (nextTab && nextTab !== activeTab) {
                setActiveTab(nextTab)
            }
        }

        window.addEventListener('popstate', handlePopState)
        return () => window.removeEventListener('popstate', handlePopState)
    }, [activeTab])

    useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        params.set(TAB_PARAM, activeTab)
        const nextQuery = params.toString()
        const nextUrl = nextQuery ? `${window.location.pathname}?${nextQuery}` : window.location.pathname
        window.history.replaceState(null, '', nextUrl)
    }, [activeTab])

    return (
        <section className='animate space-y-3'>
            {!auth.user ? (
                <div className='border border-warning/40 bg-warning/10 p-4 text-sm text-warning'>{t('admin.loginRequired')}</div>
            ) : auth.user.role !== 'admin' ? (
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{t('admin.accessDenied')}</div>
            ) : (
                <>
                    <div className='lg:hidden'>
                        <select className='w-full border border-border px-3 py-2 text-sm text-text focus:border-accent focus:outline-none' value={activeTab} onChange={(event) => setActiveTab(event.target.value as AdminTabId)}>
                            {adminTabs.map((tab) => (
                                <option key={tab.id} value={tab.id}>
                                    {tab.label}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div className='flex flex-col gap-4 lg:flex-row'>
                        <nav className='hidden w-64 shrink-0 lg:block'>
                            {adminTabs.map((tab) => (
                                <button
                                    key={tab.id}
                                    className={`flex w-full items-center rounded-none border-b border-border px-4 py-3 text-left text-sm ${activeTab === tab.id ? 'bg-surface-muted font-semibold text-accent' : 'text-text-muted hover:bg-surface-muted'}`}
                                    onClick={() => setActiveTab(tab.id as AdminTabId)}
                                    type='button'
                                >
                                    {tab.label}
                                </button>
                            ))}
                        </nav>

                        <div className='min-w-0 flex-1'>
                            {activeTab === 'challenge_create' ? (
                                <CreateChallenge />
                            ) : activeTab === 'challenge_management' ? (
                                <ChallengeManagement />
                            ) : activeTab === 'stacks' ? (
                                <Stacks />
                            ) : activeTab === 'users' ? (
                                <Users />
                            ) : activeTab === 'affiliations' ? (
                                <Affiliations />
                            ) : null}
                        </div>
                    </div>
                </>
            )}
        </section>
    )
}

export default Admin
