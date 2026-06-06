import { useCallback, useEffect, useState } from 'react'
import { uploadPresignedPost } from '../../lib/api'
import { useApi } from '../../lib/useApi'
import { useT } from '../../lib/i18n'
import { formatApiError } from '../../lib/utils'
import { mediaURL } from '../../lib/media'
import type { Popup } from '../../lib/types'
import FormMessage from '../../components/FormMessage'

const AdminPopups = () => {
    const api = useApi()
    const t = useT()
    const [rows, setRows] = useState<Popup[]>([])
    const [title, setTitle] = useState('')
    const [linkURL, setLinkURL] = useState('')
    const [active, setActive] = useState(false)
    const [createImage, setCreateImage] = useState<File | null>(null)
    const [loading, setLoading] = useState(false)
    const [saving, setSaving] = useState(false)
    const [uploadingID, setUploadingID] = useState<number | null>(null)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')

    const loadRows = useCallback(async () => {
        setLoading(true)
        setErrorMessage('')
        try {
            const data = await api.adminPopups()
            setRows(data.popups)
        } catch (error) {
            setRows([])
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setLoading(false)
        }
    }, [api, t])

    useEffect(() => {
        void loadRows()
    }, [loadRows])

    const replaceRow = (popup: Popup) => {
        setRows((prev) => prev.map((row) => (row.id === popup.id ? popup : row)))
    }

    const uploadPopupImage = async (popup: Popup, file: File) => {
        if (!file.type.startsWith('image/')) {
            throw new Error(t('admin.popups.imageTypeError'))
        }

        setUploadingID(popup.id)
        const upload = await api.requestPopupImageUpload(popup.id, file.name)
        await uploadPresignedPost(upload.upload, file)
        return api.finalizePopupImageUpload(popup.id, upload.upload.fields?.key ?? '', file.name)
    }

    const createPopup = async () => {
        const trimmed = title.trim()
        if (!trimmed) {
            setErrorMessage(t('errors.required'))
            return
        }
        if (active && !createImage) {
            setErrorMessage(t('admin.popups.activeRequiresImage'))
            return
        }

        setSaving(true)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            const trimmedLink = linkURL.trim()
            let created = await api.createPopup({ title: trimmed, link_url: trimmedLink || null, is_active: false })
            setRows((prev) => [created, ...prev])
            if (createImage) {
                created = await uploadPopupImage(created, createImage)
                if (active) created = await api.updatePopup(created.id, { is_active: true })
                setRows((prev) => prev.map((row) => (row.id === created.id ? created : row)))
            }
            setTitle('')
            setLinkURL('')
            setActive(false)
            setCreateImage(null)
            setSuccessMessage(t('admin.popups.created'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSaving(false)
            setUploadingID(null)
        }
    }

    const updatePopup = async (popup: Popup, patch: { title?: string; link_url?: string | null; is_active?: boolean }) => {
        setSaving(true)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            const updated = await api.updatePopup(popup.id, patch)
            replaceRow(updated)
            setSuccessMessage(t('admin.popups.updated'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSaving(false)
        }
    }

    const deletePopup = async (popup: Popup) => {
        if (!window.confirm(t('admin.popups.deleteConfirm'))) return

        setSaving(true)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            await api.deletePopup(popup.id)
            setRows((prev) => prev.filter((row) => row.id !== popup.id))
            setSuccessMessage(t('admin.popups.deleted'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSaving(false)
        }
    }

    const uploadImage = async (popup: Popup, file: File | null) => {
        if (!file) return
        if (!file.type.startsWith('image/')) {
            setErrorMessage(t('admin.popups.imageTypeError'))
            return
        }

        setUploadingID(popup.id)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            const updated = await uploadPopupImage(popup, file)
            replaceRow(updated)
            setSuccessMessage(t('admin.popups.imageUploaded'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setUploadingID(null)
        }
    }

    const deleteImage = async (popup: Popup) => {
        setUploadingID(popup.id)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            const updated = await api.deletePopupImage(popup.id)
            replaceRow(updated)
            setSuccessMessage(t('admin.popups.imageDeleted'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setUploadingID(null)
        }
    }

    return (
        <section className='space-y-4'>
            <div className='space-y-3 rounded-lg border border-border bg-surface p-4'>
                <h3 className='text-base text-text'>{t('admin.popups.title')}</h3>
                <div className='grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_auto_auto] md:items-center'>
                    <input className='w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none' value={title} onChange={(event) => setTitle(event.target.value)} placeholder={t('admin.popups.titlePlaceholder')} disabled={saving} />
                    <input className='w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none' value={linkURL} onChange={(event) => setLinkURL(event.target.value)} placeholder={t('admin.popups.linkPlaceholder')} disabled={saving} />
                    <label className='rounded-md border border-border bg-surface-muted px-4 py-2 text-sm text-text hover:bg-surface-subtle'>
                        <input
                            className='sr-only'
                            type='file'
                            accept='image/png,image/jpeg,image/webp'
                            disabled={saving}
                            onChange={(event) => {
                                const file = event.target.files?.[0] ?? null
                                setCreateImage(file)
                                if (!file) setActive(false)
                            }}
                        />
                        {createImage ? createImage.name : t('admin.popups.selectImage')}
                    </label>
                    <label className='flex items-center gap-2 text-sm text-text'>
                        <input type='checkbox' checked={active} onChange={(event) => setActive(event.target.checked)} disabled={saving || !createImage} />
                        {t('admin.popups.active')}
                    </label>
                    <button type='button' className='rounded-md border border-border bg-surface-muted px-4 py-2 text-sm text-text hover:bg-surface-subtle disabled:opacity-50' disabled={saving} onClick={() => void createPopup()}>
                        {saving ? t('admin.popups.saving') : t('admin.popups.create')}
                    </button>
                </div>
            </div>

            {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
            {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

            <div className='rounded-lg border border-border bg-surface'>
                <div className='grid grid-cols-[80px_minmax(0,1fr)_110px] gap-3 border-b border-border bg-surface-muted px-4 py-2 text-xs text-text-muted md:grid-cols-[80px_160px_minmax(0,1fr)_120px_160px]'>
                    <span>{t('common.id')}</span>
                    <span className='hidden md:block'>{t('admin.popups.preview')}</span>
                    <span>{t('admin.popups.titleColumn')}</span>
                    <span>{t('admin.popups.status')}</span>
                    <span className='hidden md:block'>{t('admin.popups.actions')}</span>
                </div>

                {loading ? (
                    <p className='px-4 py-6 text-center text-sm text-text-muted'>{t('common.loading')}</p>
                ) : rows.length === 0 ? (
                    <p className='px-4 py-6 text-center text-sm text-text-muted'>{t('admin.popups.empty')}</p>
                ) : (
                    <div className='divide-y divide-border/70'>
                        {rows.map((popup) => {
                            const imageURL = mediaURL(popup.image_key)
                            const busy = saving || uploadingID === popup.id

                            return (
                                <div key={popup.id} className='grid grid-cols-[80px_minmax(0,1fr)_110px] gap-3 px-4 py-4 text-sm text-text md:grid-cols-[80px_160px_minmax(0,1fr)_120px_160px]'>
                                    <span className='pt-2 text-text-muted'>{popup.id}</span>
                                    <div className='hidden md:block'>
                                        <div className='w-24 overflow-hidden rounded-md border border-border bg-white' style={{ aspectRatio: '210 / 297' }}>{imageURL ? <img className='h-full w-full object-contain' src={imageURL} alt={popup.title} /> : null}</div>
                                    </div>
                                    <div className='space-y-2'>
                                        <input className='w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none' defaultValue={popup.title} disabled={busy} onBlur={(event) => {
                                            const nextTitle = event.target.value.trim()
                                            if (nextTitle && nextTitle !== popup.title) void updatePopup(popup, { title: nextTitle })
                                        }} />
                                        <input
                                            className='w-full rounded-md border border-border bg-surface px-3 py-2 text-xs text-text focus:border-accent focus:outline-none'
                                            defaultValue={popup.link_url ?? ''}
                                            disabled={busy}
                                            placeholder={t('admin.popups.linkPlaceholder')}
                                            onBlur={(event) => {
                                                const nextLink = event.target.value.trim()
                                                const currentLink = popup.link_url ?? ''
                                                if (nextLink !== currentLink) void updatePopup(popup, { link_url: nextLink || null })
                                            }}
                                        />
                                        <div className='flex flex-wrap items-center gap-2'>
                                            <label className='rounded-md border border-border bg-surface-muted px-3 py-1.5 text-xs text-text hover:bg-surface-subtle'>
                                                <input className='sr-only' type='file' accept='image/png,image/jpeg,image/webp' disabled={busy} onChange={(event) => void uploadImage(popup, event.target.files?.[0] ?? null)} />
                                                {uploadingID === popup.id ? t('admin.popups.uploading') : t('admin.popups.uploadImage')}
                                            </label>
                                            <button type='button' className='rounded-md border border-border px-3 py-1.5 text-xs text-text disabled:opacity-50' disabled={busy || !popup.image_key} onClick={() => void deleteImage(popup)}>
                                                {t('admin.popups.deleteImage')}
                                            </button>
                                            <button type='button' className='rounded-md border border-danger/50 px-3 py-1.5 text-xs text-danger disabled:opacity-50 md:hidden' disabled={busy} onClick={() => void deletePopup(popup)}>
                                                {t('common.delete')}
                                            </button>
                                        </div>
                                        {popup.image_name ? <p className='truncate text-xs text-text-muted'>{popup.image_name}</p> : null}
                                    </div>
                                    <label className='flex items-start gap-2 pt-2 text-xs text-text'>
                                        <input type='checkbox' checked={popup.is_active} disabled={busy || (!popup.image_key && !popup.is_active)} onChange={(event) => void updatePopup(popup, { is_active: event.target.checked })} />
                                        {popup.is_active ? t('admin.popups.active') : t('admin.popups.inactive')}
                                    </label>
                                    <div className='hidden items-start md:flex'>
                                        <button type='button' className='rounded-md border border-danger/50 px-3 py-1.5 text-xs text-danger disabled:opacity-50' disabled={busy} onClick={() => void deletePopup(popup)}>
                                            {t('common.delete')}
                                        </button>
                                    </div>
                                </div>
                            )
                        })}
                    </div>
                )}
            </div>
        </section>
    )
}

export default AdminPopups
