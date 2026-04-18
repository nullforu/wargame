import { useEffect, useRef, useState } from 'react'
import { uploadPresignedPost } from '../../lib/api'
import { CHALLENGE_CATEGORIES } from '../../lib/constants'
import { formatApiError, isZipFile, type FieldErrors } from '../../lib/utils'
import MonacoEditor from '../../components/MonacoEditor'
import FormMessage from '../../components/FormMessage'
import { getCategoryKey, useT } from '../../lib/i18n'
import { useApi } from '../../lib/useApi'
import type { Challenge, TargetPortSpec } from '../../lib/types'

type TargetPortRow = TargetPortSpec & { id: string }

const CreateChallenge = () => {
    const t = useT()
    const api = useApi()
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')
    const [title, setTitle] = useState('Example title: Enter a concise and clear title')
    const [description, setDescription] = useState('')
    const [category, setCategory] = useState<string>(CHALLENGE_CATEGORIES[0])
    const [points, setPoints] = useState(100)
    const [minimumPoints, setMinimumPoints] = useState(100)
    const [flag, setFlag] = useState('')
    const [isActive, setIsActive] = useState(true)
    const [previousChallengeId, setPreviousChallengeId] = useState<number | ''>('')
    const [stackEnabled, setStackEnabled] = useState(false)
    const portIdRef = useRef(0)
    const newPortRow = (port?: TargetPortSpec): TargetPortRow => ({
        id: `port-${portIdRef.current++}`,
        container_port: port?.container_port ?? 80,
        protocol: port?.protocol ?? 'TCP',
    })
    const [stackTargetPorts, setStackTargetPorts] = useState<TargetPortRow[]>([newPortRow()])
    const [stackPodSpec, setStackPodSpec] = useState('')
    const [challengeFile, setChallengeFile] = useState<File | null>(null)
    const [challengeFileError, setChallengeFileError] = useState('')
    const [challengeFileUploading, setChallengeFileUploading] = useState(false)
    const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
    const [availableChallenges, setAvailableChallenges] = useState<Challenge[]>([])
    const fileInputRef = useRef<HTMLInputElement | null>(null)
    const sortedChallenges = [...availableChallenges].sort((a, b) => a.id - b.id)
    const formatChallengeOption = (challenge: Challenge) => {
        const categoryValue = 'category' in challenge && challenge.category ? challenge.category : t('common.na')
        return `#${challenge.id} ${challenge.title} (${t(getCategoryKey(categoryValue))})`
    }

    useEffect(() => {
        const loadChallenges = async () => {
            try {
                const data = await api.challenges()
                setAvailableChallenges(data.challenges)
            } catch (error) {
                const formatted = formatApiError(error, t)
                setErrorMessage(formatted.message)
            }
        }

        void loadChallenges()
    }, [api, t])

    const submit = async () => {
        setLoading(true)
        setErrorMessage('')
        setSuccessMessage('')
        setFieldErrors({})
        setChallengeFileError('')

        try {
            if (challengeFile && !isZipFile(challengeFile)) {
                setChallengeFileError(t('admin.create.onlyZip'))
                return
            }

            const created = await api.createChallenge({
                title,
                description,
                category,
                points: Number(points),
                minimum_points: Number(minimumPoints),
                flag,
                is_active: isActive,
                previous_challenge_id: previousChallengeId === '' ? undefined : Number(previousChallengeId),
                stack_enabled: stackEnabled,
                stack_target_ports: stackEnabled
                    ? stackTargetPorts.map(({ container_port, protocol }) => ({
                          container_port,
                          protocol,
                      }))
                    : undefined,
                stack_pod_spec: stackEnabled ? stackPodSpec : undefined,
            })

            setSuccessMessage(t('admin.create.success', { title: created.title, id: created.id }))

            if (challengeFile) {
                try {
                    setChallengeFileUploading(true)
                    const uploadResponse = await api.requestChallengeFileUpload(created.id, challengeFile.name)
                    await uploadPresignedPost(uploadResponse.upload, challengeFile)
                    setSuccessMessage(t('admin.create.successWithFile', { title: created.title, id: created.id }))
                } catch (uploadError) {
                    const formattedUpload = formatApiError(uploadError, t)
                    setErrorMessage(t('admin.create.fileUploadFailed', { message: formattedUpload.message }))
                } finally {
                    setChallengeFileUploading(false)
                }
            }

            setTitle('')
            setDescription('')
            setCategory(CHALLENGE_CATEGORIES[0])
            setPoints(100)
            setMinimumPoints(100)
            setFlag('')
            setIsActive(true)
            setPreviousChallengeId('')
            setChallengeFile(null)
            setStackEnabled(false)
            setStackTargetPorts([newPortRow()])
            setStackPodSpec('')

            if (fileInputRef.current) {
                fileInputRef.current.value = ''
            }
        } catch (error) {
            const formatted = formatApiError(error, t)
            setErrorMessage(formatted.message)
            setFieldErrors(formatted.fieldErrors)
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className='space-y-4'>
            <div className='rounded-3xl border border-border bg-surface p-4 md:p-8'>
                <form
                    className='space-y-5'
                    onSubmit={(event) => {
                        event.preventDefault()
                        submit()
                    }}
                >
                    <div>
                        <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-title'>
                            {t('common.title')}
                        </label>
                        <input
                            id='admin-title'
                            className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                            type='text'
                            value={title}
                            onChange={(event) => setTitle(event.target.value)}
                        />
                        {fieldErrors.title ? (
                            <p className='mt-2 text-xs text-danger'>
                                {t('common.title')}: {fieldErrors.title}
                            </p>
                        ) : null}
                    </div>
                    <div>
                        <p className='text-xs uppercase tracking-wide text-text-muted'>{t('common.description')}</p>
                        <div className='mt-2 w-full rounded-xl border border-border bg-surface py-4 text-sm text-text focus-within:border-accent'>
                            <MonacoEditor template='markdown' language='markdown' value={description} onChange={(value) => setDescription(value)} />
                        </div>
                        {fieldErrors.description ? (
                            <p className='mt-2 text-xs text-danger'>
                                {t('common.description')}: {fieldErrors.description}
                            </p>
                        ) : null}
                    </div>
                    <div className='grid gap-4 md:grid-cols-3'>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-category'>
                                {t('common.category')}
                            </label>
                            <select
                                id='admin-category'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                value={category}
                                onChange={(event) => setCategory(event.target.value)}
                            >
                                {CHALLENGE_CATEGORIES.map((option) => (
                                    <option key={option} value={option}>
                                        {t(getCategoryKey(option))}
                                    </option>
                                ))}
                            </select>
                            {fieldErrors.category ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('common.category')}: {fieldErrors.category}
                                </p>
                            ) : null}
                        </div>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-points'>
                                {t('common.points')}
                            </label>
                            <input
                                id='admin-points'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                type='number'
                                min={1}
                                value={points}
                                onChange={(event) => setPoints(Number(event.target.value))}
                            />
                            {fieldErrors.points ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('common.points')}: {fieldErrors.points}
                                </p>
                            ) : null}
                        </div>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-minimum-points'>
                                {t('common.minimum')}
                            </label>
                            <input
                                id='admin-minimum-points'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                type='number'
                                min={0}
                                value={minimumPoints}
                                onChange={(event) => setMinimumPoints(Number(event.target.value))}
                            />
                            {fieldErrors.minimum_points ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('common.minimum')}: {fieldErrors.minimum_points}
                                </p>
                            ) : null}
                        </div>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-flag'>
                                {t('common.flag')}
                            </label>
                            <input
                                id='admin-flag'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                type='text'
                                value={flag}
                                onChange={(event) => setFlag(event.target.value)}
                            />
                            {fieldErrors.flag ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('common.flag')}: {fieldErrors.flag}
                                </p>
                            ) : null}
                        </div>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-previous-challenge'>
                                {t('admin.create.previousChallenge')}
                            </label>
                            <select
                                id='admin-previous-challenge'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                value={previousChallengeId === '' ? '' : String(previousChallengeId)}
                                onChange={(event) => {
                                    const value = event.target.value
                                    setPreviousChallengeId(value === '' ? '' : Number(value))
                                }}
                            >
                                <option value=''>{t('admin.create.previousChallengeNone')}</option>
                                {sortedChallenges.map((challenge) => (
                                    <option key={challenge.id} value={challenge.id}>
                                        {formatChallengeOption(challenge)}
                                    </option>
                                ))}
                            </select>
                            {fieldErrors.previous_challenge_id ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('admin.create.previousChallenge')}: {fieldErrors.previous_challenge_id}
                                </p>
                            ) : null}
                        </div>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-file'>
                                {t('admin.create.challengeFile')}
                            </label>
                            <input
                                id='admin-file'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                type='file'
                                accept='.zip'
                                ref={fileInputRef}
                                onChange={(event) => {
                                    const target = event.currentTarget
                                    setChallengeFile(target.files?.[0] ?? null)
                                    setChallengeFileError('')
                                }}
                            />
                            {challengeFileError ? <p className='mt-2 text-xs text-danger'>{challengeFileError}</p> : null}
                        </div>
                    </div>
                    <label className='flex items-center gap-3 text-sm text-text'>
                        <input type='checkbox' checked={isActive} onChange={(event) => setIsActive(event.target.checked)} className='h-4 w-4 rounded border-border' />
                        {t('admin.create.createActive')}
                    </label>
                    <div className='rounded-2xl border border-border bg-surface/60 p-4'>
                        <label className='flex items-center gap-3 text-sm text-text'>
                            <input type='checkbox' checked={stackEnabled} onChange={(event) => setStackEnabled(event.target.checked)} className='h-4 w-4 rounded border-border' />
                            {t('admin.create.provideStack')}
                        </label>
                        {stackEnabled ? (
                            <div className='mt-4 grid gap-4'>
                                <div>
                                    <div className='flex flex-wrap items-center justify-between gap-2'>
                                        <label className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.create.targetPorts')}</label>
                                        <button
                                            className='text-xs text-accent hover:underline disabled:opacity-60 cursor-pointer'
                                            type='button'
                                            onClick={() => setStackTargetPorts((prev) => (prev.length >= 24 ? prev : [...prev, newPortRow()]))}
                                            disabled={stackTargetPorts.length >= 24}
                                        >
                                            {t('common.add')}
                                        </button>
                                    </div>
                                    <div className='mt-3 grid gap-3'>
                                        {stackTargetPorts.map((port, index) => (
                                            <div key={port.id} className='grid gap-3 sm:grid-cols-[1fr_120px_auto] items-center'>
                                                <input
                                                    className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                    type='number'
                                                    min={1}
                                                    max={65535}
                                                    value={port.container_port}
                                                    onChange={(event) => {
                                                        const value = Number(event.target.value)
                                                        setStackTargetPorts((prev) => prev.map((item, idx) => (idx === index ? { ...item, container_port: value } : item)))
                                                    }}
                                                />
                                                <select
                                                    className='w-full min-w-22.5 rounded-xl border border-border bg-surface px-3 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                                    value={port.protocol}
                                                    onChange={(event) => {
                                                        const value = event.target.value as TargetPortSpec['protocol']
                                                        setStackTargetPorts((prev) => prev.map((item, idx) => (idx === index ? { ...item, protocol: value } : item)))
                                                    }}
                                                >
                                                    <option value='TCP'>TCP</option>
                                                    <option value='UDP'>UDP</option>
                                                </select>
                                                <button
                                                    className='min-w-18 rounded-lg border border-border px-3 py-2 text-xs text-text transition hover:border-border disabled:opacity-60 cursor-pointer'
                                                    type='button'
                                                    onClick={() => setStackTargetPorts((prev) => prev.filter((_, idx) => idx !== index))}
                                                    disabled={stackTargetPorts.length <= 1}
                                                >
                                                    {t('common.remove')}
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                    {fieldErrors.stack_target_ports ? (
                                        <p className='mt-2 text-xs text-danger'>
                                            {t('admin.create.targetPorts')}: {fieldErrors.stack_target_ports}
                                        </p>
                                    ) : null}
                                    {stackTargetPorts.length >= 24 ? <p className='mt-2 text-xs text-text-muted'>{t('admin.create.maxPorts')}</p> : null}
                                </div>
                                <div>
                                    <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='admin-stack-pod-spec'>
                                        {t('admin.create.podSpec')}
                                    </label>
                                    <div className='mt-2 w-full rounded-xl border border-border bg-surface py-4 text-sm text-text focus-within:border-accent'>
                                        <MonacoEditor template='yaml' language='yaml' value={stackPodSpec} onChange={(value) => setStackPodSpec(value)} />
                                    </div>
                                    {fieldErrors.stack_pod_spec ? (
                                        <p className='mt-2 text-xs text-danger'>
                                            {t('admin.create.podSpec')}: {fieldErrors.stack_pod_spec}
                                        </p>
                                    ) : null}
                                </div>
                            </div>
                        ) : null}
                    </div>

                    {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
                    {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

                    <button className='w-full rounded-xl bg-accent py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer' type='submit' disabled={loading || challengeFileUploading}>
                        {loading ? t('auth.creating') : challengeFileUploading ? t('admin.create.uploading') : t('admin.create.createChallenge')}
                    </button>
                </form>
            </div>
        </div>
    )
}

export default CreateChallenge
