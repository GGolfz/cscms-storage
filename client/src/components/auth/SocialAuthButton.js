import Icon from '../util/Icon'
import styles from '../../styles/auth/SocialAuth.module.css'
import { toTitleCase } from '../../utils/formatText'
const SocialAuthButton = ({ mode, platform }) => {
	const handleAction = (mode, platform) => {
		window.location.href = `https://storage.cscms.me/auth/${platform.toLowerCase()}`
	}
	return (
		<div
			className={`${styles.SocialBtn} ${styles[platform]}`}
			onClick={() => handleAction(mode, platform)}
		>
			<Icon name={platform} />
			<span>
				{toTitleCase(mode)} with {platform}
			</span>
		</div>
	)
}
export default SocialAuthButton
