import { Fragment } from 'react'
import FileIcon from '../util/FileIcon'
import { formatFileSize } from '../../utils/formatFileSize'
import { formatDate } from '../../utils/formatText'
import Icon from '../util/Icon'
import styles from '../../styles/file/Detail.module.css'
import Swal from 'sweetalert2'
import axios from 'axios'

const ImageDetail = ({fetchFiles, file, setError }) => {
	const copyToClipboard = () => {
		var copyText = document.createElement('input')
		copyText.setAttribute('value', file.url)
		document.body.appendChild(copyText)
		copyText.select()
		copyText.setSelectionRange(0, 99999)
		navigator.clipboard.writeText(copyText.value)
		document.body.removeChild(copyText)
	}
	const handleDelete = () => {
		axios
			.delete(`https://storage.cscms.me/api/image/${file.id}`)
			.then(() => {
				fetchFiles();
				Swal.fire({
					title: 'Deleted!',
					text: 'Your image was successfully deleted',
					type: 'success'
				})
			})
			.catch(err => {
				setError(err.response.data.message)
			})
	}
	const handleClickDelete = () => {
		Swal.fire({
			title: 'Do you want to delete image?',
			text: "You won't be able to revert this!",
			confirmButtonText: 'Delete',
			showCancelButton: true,
			reverseButtons: true,
			cancelButtonText: 'Cancel',
			confirmButtonColor: '#dc3545'
		}).then(result => {
			if (result.isConfirmed) {
				handleDelete()
			}
		})
	}
	return (
		<Fragment>
			<td>
				<FileIcon ext={file.filename.split('.')[1]} type={file.file_type} />{' '}
				{file.filename}
			</td>
			<td>{formatFileSize(file.file_size)}</td>
			<td>{formatDate(file.updated_at)}</td>
			<td>
				<div className={styles.ActionList}>
					<div onClick={handleClickDelete}>
						<Icon name="delete" />
					</div>
					<div onClick={copyToClipboard}>
						<Icon name="copy" />
					</div>
				</div>
			</td>
		</Fragment>
	)
}

export default ImageDetail
