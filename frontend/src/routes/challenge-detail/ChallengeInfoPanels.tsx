import UserAvatar from '../../components/UserAvatar'
import { navigate } from '../../lib/router'
import type { Challenge, ChallengeSolver, PaginationMeta } from '../../lib/types'

interface ChallengeInfoPanelsProps {
    challenge: Challenge
    firstBloodSolver: ChallengeSolver | null
    firstBloodSolvedAfterLabel: string | null
    solvers: ChallengeSolver[]
    solverPagination: PaginationMeta
    solverPage: number
    formatTimestamp: (value: string) => string
    onSetSolverPage: (page: number) => void
    onPushSolverPageQuery: (page: number) => void
    t: (key: string, vars?: Record<string, string | number>) => string
}

const ChallengeInfoPanels = ({ challenge, firstBloodSolver, firstBloodSolvedAfterLabel, solvers, solverPagination, solverPage, formatTimestamp, onSetSolverPage, onPushSolverPageQuery, t }: ChallengeInfoPanelsProps) => {
    const creatorName = challenge.created_by?.username?.trim()
    const creatorAffiliation = challenge.created_by?.affiliation?.trim()
    const creatorBio = challenge.created_by?.bio?.trim()

    return (
        <>
            <section className='space-y-3 px-1'>
                <h2 className='text-xl font-semibold text-text'>{t('challenges.tableAuthor')}</h2>

                <div className='rounded-2xl bg-surface/70'>
                    {creatorName ? (
                        <div className='flex items-start justify-between gap-4 py-2'>
                            <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                <UserAvatar username={creatorName} size='md' />
                                <div className='min-w-0'>
                                    {challenge.created_by?.user_id ? (
                                        <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${challenge.created_by?.user_id}`)}>
                                            {creatorName}
                                        </button>
                                    ) : (
                                        <div className='block max-w-full truncate text-left text-lg font-semibold text-text'>{creatorName}</div>
                                    )}
                                    <p className='mt-1 text-sm text-text-subtle'>{creatorAffiliation ? creatorAffiliation : ''}</p>
                                    <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{creatorBio && creatorBio.length > 0 ? creatorBio : t('profile.noBio')}</p>
                                </div>
                            </div>
                        </div>
                    ) : (
                        <div className='flex items-start justify-between gap-4 py-2'>
                            <div className='min-w-0 flex-1'>
                                <p className='text-lg font-semibold text-text'>{t('common.na')}</p>
                                <p className='mt-1 text-sm text-text-subtle'></p>
                            </div>
                        </div>
                    )}
                </div>
            </section>

            {firstBloodSolver ? (
                <section className='space-y-3 px-1'>
                    <h2 className='flex items-center gap-2 text-xl font-semibold text-danger'>
                        <svg viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg' className='h-5 w-5'>
                            <path d='M5 6.7c.9-.8 2.1-1.2 3.5-1.2 2.7 0 4.6 2.2 8.5.6v8.8c-3.9 1.7-5.8-.9-8.5-.9-1.2 0-2.5.3-3.5.9V6.7Z' fill='currentColor' opacity='0.2' />
                            <path
                                d='M4.5 21V16M4.5 16V6.5C5.5 5.5 7 5 8.5 5C11.5 5 13.5 7.5 17.5 5.5V15.5C13.5 17.5 11.5 14.5 8.5 14.5C7.5 14.5 5.5 15 4.5 16Z'
                                fill='none'
                                stroke='currentColor'
                                strokeLinecap='round'
                                strokeLinejoin='round'
                            />
                        </svg>
                        {t('leaderboard.firstBlood')}
                    </h2>

                    <div className='rounded-2xl bg-surface/70'>
                        <div className='flex items-start justify-between gap-4 py-2'>
                            <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                <UserAvatar username={firstBloodSolver.username} size='md' />
                                <div className='min-w-0'>
                                    <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${firstBloodSolver.user_id}`)}>
                                        {firstBloodSolver.username}
                                    </button>
                                    <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>
                                        {firstBloodSolver.affiliation && firstBloodSolver.affiliation.trim().length > 0 ? `${firstBloodSolver.affiliation} · ` : ''}
                                        {firstBloodSolver.bio && firstBloodSolver.bio.trim().length > 0 ? firstBloodSolver.bio : t('profile.noBio')}
                                    </p>
                                    <p className='mt-1 text-sm text-danger'>{firstBloodSolvedAfterLabel ?? formatTimestamp(firstBloodSolver.solved_at)}</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </section>
            ) : null}

            <section className='space-y-3 px-1'>
                <h2 className='text-xl font-semibold text-text'>{t('challenge.recentSolversTitle')}</h2>

                <div className='space-y-3'>
                    {solvers.length === 0 ? (
                        <p className='text-sm text-text-muted'>{t('challenge.noSolversYet')}</p>
                    ) : (
                        solvers.map((solver, index) => (
                            <div key={`${solver.user_id}-${index}`} className='flex items-start justify-between gap-4 py-2'>
                                <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                    <UserAvatar username={solver.username} size='md' />
                                    <div className='min-w-0'>
                                        <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${solver.user_id}`)}>
                                            {solver.username}
                                        </button>

                                        <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>
                                            {solver.affiliation && solver.affiliation.trim().length > 0 ? `${solver.affiliation} · ` : ''}
                                            {solver.bio && solver.bio.trim().length > 0 ? solver.bio : t('profile.noBio')}
                                        </p>
                                        <p className='mt-1 text-sm text-text-subtle'>{formatTimestamp(solver.solved_at)}</p>
                                    </div>
                                </div>

                                <span className='shrink-0 text-sm text-text-subtle'>{index + 1}</span>
                            </div>
                        ))
                    )}
                </div>

                <div className='flex items-center justify-between pt-2 text-sm text-text-muted'>
                    <span>
                        {solverPagination.page} / {solverPagination.total_pages || 1}
                    </span>

                    <div className='flex gap-2'>
                        <button
                            className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                            disabled={!solverPagination.has_prev}
                            onClick={() => {
                                const next = Math.max(1, solverPage - 1)
                                onSetSolverPage(next)
                                onPushSolverPageQuery(next)
                            }}
                        >
                            {t('common.previous')}
                        </button>

                        <button
                            className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                            disabled={!solverPagination.has_next}
                            onClick={() => {
                                const next = solverPage + 1
                                onSetSolverPage(next)
                                onPushSolverPageQuery(next)
                            }}
                        >
                            {t('common.next')}
                        </button>
                    </div>
                </div>
            </section>
        </>
    )
}

export default ChallengeInfoPanels
