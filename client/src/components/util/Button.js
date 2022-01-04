import styles from '../../styles/util/Button.module.css'
const Button = ({ children, className, color, bgColor, action, style }) => {
	return (
		<button className={styles.Btn + ' ' + className} style={{ color: color, background: bgColor, ...style }} onClick={action}>
			{children}
		</button>
	)
}

export default Button
