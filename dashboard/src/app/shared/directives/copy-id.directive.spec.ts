import {Clipboard} from '@angular/cdk/clipboard';
import {Component} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {By} from '@angular/platform-browser';

import {SnackBarService} from '../../core';
import {provideMaterialAnimationsDisabled} from '../../core/core.testing.module';
import {CopyIdDirective} from './copy-id.directive';

@Component({
  template: `
    <span role="button" tabindex="0" [appCopyId]="id">{{id}}</span>
    <button type="button" [appCopyId]="id">Copy</button>
  `,
  imports: [CopyIdDirective],
  standalone: true,
})
class TestHostComponent {
  id = 'config-1';
}

describe('CopyIdDirective', () => {
  let fixture: ComponentFixture<TestHostComponent>;
  let copy: ReturnType<typeof vi.fn>;
  let openSnackBar: ReturnType<typeof vi.fn>;
  let openError: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    copy = vi.fn(() => true);
    openSnackBar = vi.fn();
    openError = vi.fn();

    await TestBed.configureTestingModule({
      imports: [TestHostComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {provide: Clipboard, useValue: {copy}},
        {provide: SnackBarService, useValue: {openSnackBar, openError}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    fixture.detectChanges();
  });

  it('copies IDs by click and announces success', () => {
    fixture.nativeElement.querySelector('span').click();

    expect(copy).toHaveBeenCalledWith('config-1');
    expect(openSnackBar).toHaveBeenCalledWith('ID copied');
  });

  it('makes non-native copy controls keyboard operable without duplicating native button handling', () => {
    const copyChip = fixture.nativeElement.querySelector('span') as HTMLElement;
    const enter = new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true});
    copyChip.dispatchEvent(enter);

    expect(enter.defaultPrevented).toBe(true);
    expect(copy).toHaveBeenCalledOnce();

    const button = fixture.nativeElement.querySelector('button') as HTMLButtonElement;
    button.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(copy).toHaveBeenCalledOnce();
  });

  it('ignores empty values and reports clipboard failures', () => {
    fixture.componentInstance.id = '';
    fixture.detectChanges();
    const directive = fixture.debugElement.query(By.css('span')).injector.get(CopyIdDirective);
    directive.appCopyId = '';
    copy.mockClear();
    fixture.nativeElement.querySelector('span').click();
    expect(copy).not.toHaveBeenCalled();

    fixture.componentInstance.id = 'config-2';
    copy.mockReturnValue(false);
    fixture.detectChanges();
    fixture.nativeElement.querySelector('span').click();

    expect(openError).toHaveBeenCalledWith(expect.objectContaining({message: 'Could not copy ID'}));
  });
});
