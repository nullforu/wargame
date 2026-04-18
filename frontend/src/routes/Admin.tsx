import { useEffect, useMemo, useState } from 'react'
import CreateChallenge from './admin/CreateChallenge'
import ChallengeManagement from './admin/ChallengeManagement'
import Users from './admin/Users'
import Stacks from './admin/Stacks'
import { useT } from '../lib/i18n'
import { useAuth } from '../lib/auth'

interface RouteProps {
    routeParams?: Record<string, string>
}

type AdminTabId = 'challenge_create' | 'challenge_management' | 'users' | 'stacks'
const TAB_PARAM = 'tab'
const ADMIN_TAB_IDS: AdminTabId[] = ['challenge_create', 'challenge_management', 'users', 'stacks']

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
        <section className='animate'>
            <div className='mb-4 lg:mb-6'>
                <h2 className='text-2xl font-semibold text-text lg:text-3xl'>{t('admin.title')}</h2>
            </div>

            {!auth.user ? (
                <div className='rounded-2xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning-strong lg:p-6'>{t('admin.loginRequired')}</div>
            ) : auth.user.role !== 'admin' ? (
                <div className='rounded-2xl border border-danger/40 bg-danger/10 p-4 text-sm text-danger lg:p-6'>{t('admin.accessDenied')}</div>
            ) : (
                <>
                    <div className='mb-4 flex items-center gap-3'>
                        <select
                            className='flex-1 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none lg:hidden'
                            value={activeTab}
                            onChange={(event) => setActiveTab(event.target.value as AdminTabId)}
                        >
                            {adminTabs.map((tab) => (
                                <option key={tab.id} value={tab.id}>
                                    {tab.label}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div className='flex flex-col gap-6 lg:flex-row lg:gap-8'>
                        <nav className='hidden lg:block lg:w-64 lg:shrink-0'>
                            <div className='rounded-2xl border border-border bg-surface p-2'>
                                {adminTabs.map((tab) => (
                                    <button
                                        key={tab.id}
                                        className={`flex w-full items-center rounded-lg px-4 py-2.5 text-left text-sm transition cursor-pointer ${
                                            activeTab === tab.id ? 'bg-surface-subtle font-medium text-text' : 'text-text hover:bg-surface-muted'
                                        }`}
                                        onClick={() => setActiveTab(tab.id as AdminTabId)}
                                        type='button'
                                    >
                                        {tab.label}
                                    </button>
                                ))}
                            </div>
                        </nav>

                        <div className='flex-1 lg:min-w-0'>
                            {activeTab === 'challenge_create' ? <CreateChallenge /> : activeTab === 'challenge_management' ? <ChallengeManagement /> : activeTab === 'stacks' ? <Stacks /> : activeTab === 'users' ? <Users /> : null}
                        </div>
                    </div>
                </>
            )}
        </section>
    )
}

export default Admin
