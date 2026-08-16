package no.nb.nna.veidemann.frontier.worker;

public class IllegalSessionException extends RuntimeException {
    private static final long serialVersionUID = 1L;

    public IllegalSessionException() {
    }

    public IllegalSessionException(String message) {
        super(message);
    }

    public IllegalSessionException(String message, Throwable cause) {
        super(message, cause);
    }

    public IllegalSessionException(Throwable cause) {
        super(cause);
    }
}
