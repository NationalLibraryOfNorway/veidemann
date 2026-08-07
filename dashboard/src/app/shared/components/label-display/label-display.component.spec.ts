import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelDisplayComponent, isEmojiLabel, isSingleEmoji} from './label-display.component';

describe('LabelDisplayComponent', () => {
  let fixture: ComponentFixture<LabelDisplayComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({imports: [LabelDisplayComponent]}).compileComponents();
    fixture = TestBed.createComponent(LabelDisplayComponent);
  });

  it('recognizes one complete Unicode emoji sequence', () => {
    expect(isSingleEmoji('😀')).toBe(true);
    expect(isSingleEmoji('🇳🇴')).toBe(true);
    expect(isSingleEmoji('👍🏽')).toBe(true);
    expect(isSingleEmoji('👨‍👩‍👧‍👦')).toBe(true);
    expect(isSingleEmoji('not-an-emoji')).toBe(false);
    expect(isSingleEmoji('😀🐶')).toBe(false);
  });

  it('requires the exact lowercase emoji key', () => {
    expect(isEmojiLabel('emoji', '🐶')).toBe(true);
    expect(isEmojiLabel('Emoji', '🐶')).toBe(false);
    expect(isEmojiLabel('emoji', 'dog')).toBe(false);
  });

  it('renders a valid emoji as a Noto glyph with accessible raw text', () => {
    fixture.componentRef.setInput('key', 'emoji');
    fixture.componentRef.setInput('value', '🐶');
    fixture.detectChanges();

    const glyph = fixture.nativeElement.querySelector('.label-display__emoji') as HTMLElement;
    const accessible = fixture.nativeElement.querySelector('.label-display__accessible') as HTMLElement;
    expect(glyph.textContent).toBe('🐶');
    expect(glyph.classList).toContain('emoji-font');
    expect(glyph.getAttribute('aria-hidden')).toBe('true');
    expect(parseFloat(getComputedStyle(glyph).lineHeight)).toBeGreaterThan(1);
    expect(accessible.textContent).toBe('emoji:🐶');
  });

  it('renders normal and invalid emoji labels literally', () => {
    fixture.componentRef.setInput('key', 'emoji');
    fixture.componentRef.setInput('value', 'dog');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent.trim()).toBe('emoji:dog');
    expect(fixture.nativeElement.querySelector('.label-display__emoji')).toBeNull();
  });
});
